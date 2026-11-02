package main

import (
	"archive/zip"
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"encoding/base64"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
)

// Section markers of the binary Steam depot manifest format.
const (
	manifestPayloadMagic   = 0x71F617D0
	manifestMetadataMagic  = 0x1F4812BE
	manifestSignatureMagic = 0x1B81B817
	manifestEndMagic       = 0x32C415AB

	manifestFlagDirectory = 64
	maxManifestSize       = 512 << 20
)

// DepotManifest is the part of a Steam manifest that downgrading needs.
type DepotManifest struct {
	DepotID            uint32
	ManifestID         uint64
	FilenamesEncrypted bool
	Files              []ManifestFile
}

type ManifestFile struct {
	Name       string // exactly as stored in the manifest (decrypted when a key was supplied)
	Size       uint64
	Flags      uint32
	LinkTarget string
}

func (f ManifestFile) IsDirectory() bool { return f.Flags&manifestFlagDirectory != 0 }

// loadManifest reads a manifest file. depotKey decrypts filenames when the manifest stores them encrypted.
func loadManifest(path string, depotKey []byte) (*DepotManifest, error) {
	data, err := readManifestBytes(path)
	if err != nil {
		return nil, err
	}
	return parseManifest(data, depotKey)
}

func readManifestBytes(path string) ([]byte, error) {
	stat, err := os.Stat(path)
	if err != nil {
		return nil, err
	}
	if stat.IsDir() {
		return nil, errors.New("is a folder, not a manifest file")
	}
	if stat.Size() > maxManifestSize {
		return nil, errors.New("file is too large to be a Steam manifest")
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	// Manifests downloaded straight from the content servers are zipped around a single entry.
	if bytes.HasPrefix(data, []byte("PK")) {
		return unzipFirstEntry(data)
	}
	return data, nil
}

func unzipFirstEntry(data []byte) ([]byte, error) {
	reader, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil || len(reader.File) == 0 {
		return nil, errors.New("is not a readable Steam manifest")
	}
	entry, err := reader.File[0].Open()
	if err != nil {
		return nil, err
	}
	defer entry.Close()
	return io.ReadAll(io.LimitReader(entry, maxManifestSize))
}

func parseManifest(data []byte, depotKey []byte) (*DepotManifest, error) {
	manifest, err := parseManifestSections(data)
	if err != nil {
		return nil, err
	}
	if manifest.FilenamesEncrypted {
		if len(depotKey) == 0 {
			return nil, errors.New("has encrypted filenames and needs the depot key from keys.txt")
		}
		if err := decryptManifestNames(manifest, depotKey); err != nil {
			return nil, err
		}
	}
	return manifest, nil
}

// readManifestIdentity returns the depot and manifest IDs stored inside a manifest. It needs no depot key.
func readManifestIdentity(path string) (*DepotManifest, error) {
	data, err := readManifestBytes(path)
	if err != nil {
		return nil, err
	}
	manifest, err := parseManifestSections(data)
	if err != nil {
		return nil, err
	}
	if manifest.DepotID == 0 || manifest.ManifestID == 0 {
		return nil, errors.New("is not a valid manifest (no depot or manifest ID inside)")
	}
	manifest.Files = nil
	return manifest, nil
}

func parseManifestSections(data []byte) (*DepotManifest, error) {
	manifest := &DepotManifest{}
	var payload []byte
	sawMetadata := false
	for offset := 0; ; {
		if len(data)-offset < 4 {
			return nil, errors.New("is not a valid Steam manifest (truncated)")
		}
		magic := binary.LittleEndian.Uint32(data[offset:])
		offset += 4
		if magic == manifestEndMagic {
			break
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
		switch magic {
		case manifestPayloadMagic:
			payload = section
		case manifestMetadataMagic:
			sawMetadata = true
			if err := parseManifestMetadata(section, manifest); err != nil {
				return nil, err
			}
		case manifestSignatureMagic:
		default:
			return nil, errors.New("is not a valid Steam manifest (unknown section)")
		}
	}
	if payload == nil || !sawMetadata {
		return nil, errors.New("is not a valid Steam manifest (missing sections)")
	}
	if err := parseManifestPayload(payload, manifest); err != nil {
		return nil, err
	}
	return manifest, nil
}

func parseManifestMetadata(section []byte, manifest *DepotManifest) error {
	return walkProto(section, func(field int, value uint64, _ []byte) error {
		switch field {
		case 1:
			manifest.DepotID = uint32(value)
		case 2:
			manifest.ManifestID = value
		case 4:
			manifest.FilenamesEncrypted = value != 0
		}
		return nil
	})
}

func parseManifestPayload(section []byte, manifest *DepotManifest) error {
	return walkProto(section, func(field int, _ uint64, raw []byte) error {
		if field != 1 || raw == nil {
			return nil
		}
		var file ManifestFile
		err := walkProto(raw, func(field int, value uint64, raw []byte) error {
			switch field {
			case 1:
				file.Name = string(raw)
			case 2:
				file.Size = value
			case 3:
				file.Flags = uint32(value)
			case 7:
				file.LinkTarget = string(raw)
			}
			return nil
		})
		manifest.Files = append(manifest.Files, file)
		return err
	})
}

// decryptManifestNames decrypts every filename with the depot key.
func decryptManifestNames(manifest *DepotManifest, key []byte) error {
	for index := range manifest.Files {
		name, err := decryptName(key, manifest.Files[index].Name)
		if err != nil {
			return err
		}
		manifest.Files[index].Name = name
	}
	manifest.FilenamesEncrypted = false
	return nil
}

// decryptName mirrors SteamKit: base64, then AES-256 with an ECB-encrypted IV block followed by CBC data.
func decryptName(key []byte, encoded string) (string, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return "", errors.New("has a depot key of the wrong length (expected 64 hex characters)")
	}
	encrypted, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil || len(encrypted) < 2*aes.BlockSize || len(encrypted)%aes.BlockSize != 0 {
		return "", errors.New("could not decrypt its filenames; the depot key is probably wrong")
	}
	iv := make([]byte, aes.BlockSize)
	block.Decrypt(iv, encrypted[:aes.BlockSize])
	plain := make([]byte, len(encrypted)-aes.BlockSize)
	cipher.NewCBCDecrypter(block, iv).CryptBlocks(plain, encrypted[aes.BlockSize:])
	pad := int(plain[len(plain)-1])
	if pad < 1 || pad > aes.BlockSize || pad > len(plain) {
		return "", errors.New("could not decrypt its filenames; the depot key is probably wrong")
	}
	name := strings.TrimRight(string(plain[:len(plain)-pad]), "\x00")
	if name == "" || strings.ContainsRune(name, '�') {
		return "", errors.New("could not decrypt its filenames; the depot key is probably wrong")
	}
	return name, nil
}

// walkProto iterates the top-level fields of a protobuf message. Varint fields pass their value;
// length-delimited fields pass their bytes.
func walkProto(data []byte, visit func(field int, value uint64, raw []byte) error) error {
	for len(data) > 0 {
		tag, n := binary.Uvarint(data)
		if n <= 0 {
			return errors.New("is not a valid Steam manifest (bad field)")
		}
		data = data[n:]
		field, wire := int(tag>>3), tag&7
		switch wire {
		case 0:
			value, n := binary.Uvarint(data)
			if n <= 0 {
				return errors.New("is not a valid Steam manifest (bad number)")
			}
			data = data[n:]
			if err := visit(field, value, nil); err != nil {
				return err
			}
		case 1:
			if len(data) < 8 {
				return errors.New("is not a valid Steam manifest (truncated)")
			}
			data = data[8:]
		case 2:
			length, n := binary.Uvarint(data)
			if n <= 0 || length > uint64(len(data)-n) {
				return errors.New("is not a valid Steam manifest (bad length)")
			}
			raw := data[n : n+int(length)]
			data = data[n+int(length):]
			if err := visit(field, 0, raw); err != nil {
				return err
			}
		case 5:
			if len(data) < 4 {
				return errors.New("is not a valid Steam manifest (truncated)")
			}
			data = data[4:]
		default:
			return fmt.Errorf("is not a valid Steam manifest (field type %d)", wire)
		}
	}
	return nil
}

func protoVarint(field int, value uint64) []byte {
	return binary.AppendUvarint(binary.AppendUvarint(nil, uint64(field<<3)), value)
}

func protoBytes(field int, value []byte) []byte {
	out := binary.AppendUvarint(nil, uint64(field<<3|2))
	out = binary.AppendUvarint(out, uint64(len(value)))
	return append(out, value...)
}
