package main

import (
	"bytes"
	"crypto/sha1"
	"encoding/binary"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
)

// Compatibility mode lets a stock DepotDownloaderMod downgrade a game. The downloader already knows how to
// move a folder from the manifest recorded in .DepotDownloader/depot.config to a requested manifest, reusing
// matching chunks. DepotDrop prepares that state: both manifests are placed in .DepotDownloader (filenames
// decrypted), and the current version is recorded as installed. The "current" copy lists only files that the
// older version also has, so the downloader never deletes anything; DepotDrop removes old files itself,
// after everything is verified.

type protoElement struct {
	field   int
	wire    uint64
	raw     []byte // the element exactly as encoded
	payload []byte // contents of a length-delimited element
	value   uint64 // value of a varint element
}

// splitProto splits a protobuf message into elements without losing any bytes.
func splitProto(data []byte) ([]protoElement, error) {
	var elements []protoElement
	for len(data) > 0 {
		tag, n := binary.Uvarint(data)
		if n <= 0 {
			return nil, errors.New("is not a valid Steam manifest (bad field)")
		}
		size := n
		element := protoElement{field: int(tag >> 3), wire: tag & 7}
		rest := data[n:]
		switch element.wire {
		case 0:
			value, m := binary.Uvarint(rest)
			if m <= 0 {
				return nil, errors.New("is not a valid Steam manifest (bad number)")
			}
			element.value = value
			size += m
		case 1:
			size += 8
		case 5:
			size += 4
		case 2:
			length, m := binary.Uvarint(rest)
			if m <= 0 || length > uint64(len(rest)-m) {
				return nil, errors.New("is not a valid Steam manifest (bad length)")
			}
			size += m + int(length)
			element.payload = rest[m : m+int(length)]
		default:
			return nil, fmt.Errorf("is not a valid Steam manifest (field type %d)", element.wire)
		}
		if size > len(data) {
			return nil, errors.New("is not a valid Steam manifest (truncated)")
		}
		element.raw = data[:size]
		elements = append(elements, element)
		data = data[size:]
	}
	return elements, nil
}

// rewriteManifest returns the manifest with plain filenames. keep, when given, drops non-directory files it rejects.
func rewriteManifest(data, depotKey []byte, keep func(rel string) bool) ([]byte, error) {
	var out []byte
	for offset := 0; ; {
		if len(data)-offset < 4 {
			return nil, errors.New("is not a valid Steam manifest (truncated)")
		}
		magic := binary.LittleEndian.Uint32(data[offset:])
		offset += 4
		if magic == manifestEndMagic {
			return binary.LittleEndian.AppendUint32(out, magic), nil
		}
		if len(data)-offset < 4 {
			return nil, errors.New("is not a valid Steam manifest (truncated)")
		}
		length := int(binary.LittleEndian.Uint32(data[offset:]))
		offset += 4
		if length < 0 || length > len(data)-offset {
			return nil, errors.New("is not a valid Steam manifest (bad section size)")
		}
		section := data[offset : offset+length]
		offset += length
		var err error
		switch magic {
		case manifestMetadataMagic:
			section, err = rewriteMetadata(section)
		case manifestPayloadMagic:
			section, err = rewritePayload(section, depotKey, keep)
		}
		if err != nil {
			return nil, err
		}
		out = binary.LittleEndian.AppendUint32(out, magic)
		out = binary.LittleEndian.AppendUint32(out, uint32(len(section)))
		out = append(out, section...)
	}
}

func rewriteMetadata(section []byte) ([]byte, error) {
	elements, err := splitProto(section)
	if err != nil {
		return nil, err
	}
	var out []byte
	for _, element := range elements {
		if element.field != 4 { // filenames_encrypted: names are plain from now on
			out = append(out, element.raw...)
		}
	}
	return out, nil
}

func rewritePayload(section, depotKey []byte, keep func(rel string) bool) ([]byte, error) {
	elements, err := splitProto(section)
	if err != nil {
		return nil, err
	}
	var out []byte
	for _, element := range elements {
		if element.field != 1 || element.payload == nil {
			out = append(out, element.raw...)
			continue
		}
		inner, err := splitProto(element.payload)
		if err != nil {
			return nil, err
		}
		var name string
		var flags uint64
		for _, field := range inner {
			switch field.field {
			case 1:
				name = string(field.payload)
			case 3:
				flags = field.value
			}
		}
		if len(depotKey) > 0 {
			plain, err := decryptName(depotKey, name)
			if err != nil {
				return nil, err
			}
			name = plain
		}
		if keep != nil && flags&manifestFlagDirectory == 0 {
			rel, err := normalizeManifestPath(name)
			if err != nil {
				return nil, err
			}
			if !keep(rel) {
				continue
			}
		}
		var rebuilt []byte
		for _, field := range inner {
			if field.field == 1 {
				rebuilt = append(rebuilt, protoBytes(1, []byte(name))...)
			} else {
				rebuilt = append(rebuilt, field.raw...)
			}
		}
		out = append(out, protoBytes(1, rebuilt)...)
	}
	return out, nil
}

func writeStagedManifest(dir string, depot uint32, manifest uint64, data []byte) error {
	file := filepath.Join(dir, fmt.Sprintf("%d_%d.manifest", depot, manifest))
	if err := os.WriteFile(file, data, 0o644); err != nil {
		return err
	}
	sum := sha1.Sum(data)
	return os.WriteFile(file+".sha", sum[:], 0o644)
}

// stageDowngrade prepares .DepotDownloader so a stock downloader sees "current version installed, fetch target".
func stageDowngrade(plan *downgradePlan) error {
	dir := filepath.Join(plan.root, ".DepotDownloader")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	config := filepath.Join(dir, "depot.config")
	installed, err := readDepotConfig(config)
	if err != nil {
		installed = map[uint32]uint64{}
	}
	for _, item := range plan.depots {
		key, err := depotKeyFor(plan.request.KeysPath, item.depot.DepotID)
		if err != nil {
			return err
		}
		targetSet := map[string]bool{}
		for _, file := range item.target.Files {
			if rel, err := normalizeManifestPath(file.Name); err == nil {
				targetSet[pathKey(rel)] = true
			}
		}
		currentData, err := readManifestBytes(item.depot.CurrentManifestPath)
		if err != nil {
			return err
		}
		targetData, err := readManifestBytes(item.depot.TargetManifestPath)
		if err != nil {
			return err
		}
		trimmed, err := rewriteManifest(currentData, keyIfEncrypted(currentData, key), func(rel string) bool { return targetSet[pathKey(rel)] })
		if err != nil {
			return fmt.Errorf("the current manifest file for depot %s %w", item.depot.DepotID, err)
		}
		plain, err := rewriteManifest(targetData, keyIfEncrypted(targetData, key), nil)
		if err != nil {
			return fmt.Errorf("the previous version manifest for depot %s %w", item.depot.DepotID, err)
		}
		if bytes.Equal(trimmed, plain) {
			return errors.New("the two manifests are identical")
		}
		if err := writeStagedManifest(dir, item.current.DepotID, item.current.ManifestID, trimmed); err != nil {
			return err
		}
		if err := writeStagedManifest(dir, item.target.DepotID, item.target.ManifestID, plain); err != nil {
			return err
		}
		installed[item.current.DepotID] = item.current.ManifestID
	}
	return writeDepotConfig(config, installed)
}

// keyIfEncrypted returns the key only for manifests that store encrypted filenames.
func keyIfEncrypted(data, key []byte) []byte {
	if manifest, err := parseManifestSections(data); err == nil && manifest.FilenamesEncrypted {
		return key
	}
	return nil
}

// compatArguments is an ordinary download command; the staged state makes it a downgrade.
func compatArguments(plan *downgradePlan) [][]string {
	var commands [][]string
	for _, item := range plan.depots {
		commands = append(commands, []string{
			"-app", plan.request.AppID,
			"-depot", item.depot.DepotID,
			"-manifest", strconv.FormatUint(item.target.ManifestID, 10),
			"-depotkeys", plan.request.KeysPath,
			"-dir", plan.root,
			"-validate",
		})
	}
	return commands
}

// trimOversizeFiles cuts files that are longer than the target version says they should be.
func trimOversizeFiles(plan *downgradePlan) {
	for _, file := range plan.targets {
		if file.linked {
			continue
		}
		if full, err := resolveInside(plan.root, file.rel); err == nil {
			if stat, err := os.Stat(full); err == nil && stat.Mode().IsRegular() && uint64(stat.Size()) > file.size {
				_ = os.Truncate(full, int64(file.size))
			}
		}
	}
}

// markTargetInstalled records the target version as installed, so the next run is a plain validation of it.
func markTargetInstalled(plan *downgradePlan) error {
	config := filepath.Join(plan.root, ".DepotDownloader", "depot.config")
	installed, err := readDepotConfig(config)
	if err != nil {
		installed = map[uint32]uint64{}
	}
	for _, item := range plan.depots {
		installed[item.target.DepotID] = item.target.ManifestID
	}
	return writeDepotConfig(config, installed)
}
