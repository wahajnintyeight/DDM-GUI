export namespace main {
	
	export class DownloadRequest {
	    manifestPath: string;
	    gameFolder: string;
	    downloaderPath: string;
	    keysPath: string;
	    appId: string;
	    depotId: string;
	    validate: boolean;
	
	    static createFrom(source: any = {}) {
	        return new DownloadRequest(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.manifestPath = source["manifestPath"];
	        this.gameFolder = source["gameFolder"];
	        this.downloaderPath = source["downloaderPath"];
	        this.keysPath = source["keysPath"];
	        this.appId = source["appId"];
	        this.depotId = source["depotId"];
	        this.validate = source["validate"];
	    }
	}
	export class ManifestInfo {
	    path: string;
	    name: string;
	    depotId: string;
	    manifestId: string;
	
	    static createFrom(source: any = {}) {
	        return new ManifestInfo(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.path = source["path"];
	        this.name = source["name"];
	        this.depotId = source["depotId"];
	        this.manifestId = source["manifestId"];
	    }
	}
	export class SteamGame {
	    id: number;
	    name: string;

	    static createFrom(source: any = {}) {
	        return new SteamGame(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.name = source["name"];
	    }
	}

}

