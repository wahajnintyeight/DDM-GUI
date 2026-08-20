export namespace main {
	
	export class DownloadRequest {
	    manifestPath: string;
	    gameFolder: string;
	
	    static createFrom(source: any = {}) {
	        return new DownloadRequest(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.manifestPath = source["manifestPath"];
	        this.gameFolder = source["gameFolder"];
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

}

