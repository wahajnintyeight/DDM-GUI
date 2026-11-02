export namespace main {
	
	export class DowngradeDepot {
	    depotId: string;
	    currentManifestPath: string;
	    targetManifestPath: string;
	
	    static createFrom(source: any = {}) {
	        return new DowngradeDepot(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.depotId = source["depotId"];
	        this.currentManifestPath = source["currentManifestPath"];
	        this.targetManifestPath = source["targetManifestPath"];
	    }
	}
	export class DowngradeDepotSummary {
	    depotId: string;
	    currentManifestId: string;
	    targetManifestId: string;
	    currentFiles: number;
	    targetFiles: number;
	
	    static createFrom(source: any = {}) {
	        return new DowngradeDepotSummary(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.depotId = source["depotId"];
	        this.currentManifestId = source["currentManifestId"];
	        this.targetManifestId = source["targetManifestId"];
	        this.currentFiles = source["currentFiles"];
	        this.targetFiles = source["targetFiles"];
	    }
	}
	export class DowngradeRequest {
	    appId: string;
	    gameFolder: string;
	    downloaderPath: string;
	    keysPath: string;
	    depots: DowngradeDepot[];
	    acknowledgePartial: boolean;
	
	    static createFrom(source: any = {}) {
	        return new DowngradeRequest(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.appId = source["appId"];
	        this.gameFolder = source["gameFolder"];
	        this.downloaderPath = source["downloaderPath"];
	        this.keysPath = source["keysPath"];
	        this.depots = this.convertValues(source["depots"], DowngradeDepot);
	        this.acknowledgePartial = source["acknowledgePartial"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class DowngradeSummary {
	    appId: string;
	    gameFolder: string;
	    depots: DowngradeDepotSummary[];
	    removeCount: number;
	    removeBytes: number;
	    hadStateFolder: boolean;
	    otherDepots: string[];
	    partial: boolean;
	    notes: string[];
	
	    static createFrom(source: any = {}) {
	        return new DowngradeSummary(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.appId = source["appId"];
	        this.gameFolder = source["gameFolder"];
	        this.depots = this.convertValues(source["depots"], DowngradeDepotSummary);
	        this.removeCount = source["removeCount"];
	        this.removeBytes = source["removeBytes"];
	        this.hadStateFolder = source["hadStateFolder"];
	        this.otherDepots = source["otherDepots"];
	        this.partial = source["partial"];
	        this.notes = source["notes"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
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

