export namespace config {
	
	export class Project {
	    name: string;
	    port: number;
	    allowed_macs?: string[];
	    command?: string;
	    dir?: string;
	
	    static createFrom(source: any = {}) {
	        return new Project(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.port = source["port"];
	        this.allowed_macs = source["allowed_macs"];
	        this.command = source["command"];
	        this.dir = source["dir"];
	    }
	}

}

export namespace main {
	
	export class DirEntry {
	    name: string;
	    path: string;
	    isDir: boolean;
	    up: boolean;
	
	    static createFrom(source: any = {}) {
	        return new DirEntry(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.path = source["path"];
	        this.isDir = source["isDir"];
	        this.up = source["up"];
	    }
	}
	export class BrowseResult {
	    path: string;
	    entries: DirEntry[];
	
	    static createFrom(source: any = {}) {
	        return new BrowseResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.path = source["path"];
	        this.entries = this.convertValues(source["entries"], DirEntry);
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
	
	export class RequestsResponse {
	    stats: state.ProjStats[];
	    recent: state.ReqEntry[];
	
	    static createFrom(source: any = {}) {
	        return new RequestsResponse(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.stats = this.convertValues(source["stats"], state.ProjStats);
	        this.recent = this.convertValues(source["recent"], state.ReqEntry);
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
	export class ServerInfo {
	    ip: string;
	    proxy_port: number;
	
	    static createFrom(source: any = {}) {
	        return new ServerInfo(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.ip = source["ip"];
	        this.proxy_port = source["proxy_port"];
	    }
	}
	export class StatusRow {
	    name: string;
	    port: number;
	    up: boolean;
	    domain: string;
	    command?: string;
	    dir?: string;
	    managed: boolean;
	    latency_ms: number;
	
	    static createFrom(source: any = {}) {
	        return new StatusRow(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.port = source["port"];
	        this.up = source["up"];
	        this.domain = source["domain"];
	        this.command = source["command"];
	        this.dir = source["dir"];
	        this.managed = source["managed"];
	        this.latency_ms = source["latency_ms"];
	    }
	}

}

export namespace state {
	
	export class ProjStats {
	    name: string;
	    total: number;
	    denied: number;
	    allowed_macs: string[];
	
	    static createFrom(source: any = {}) {
	        return new ProjStats(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.total = source["total"];
	        this.denied = source["denied"];
	        this.allowed_macs = source["allowed_macs"];
	    }
	}
	export class ReqEntry {
	    // Go type: time
	    t: any;
	    project: string;
	    ip: string;
	    mac: string;
	    path: string;
	    status: number;
	
	    static createFrom(source: any = {}) {
	        return new ReqEntry(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.t = this.convertValues(source["t"], null);
	        this.project = source["project"];
	        this.ip = source["ip"];
	        this.mac = source["mac"];
	        this.path = source["path"];
	        this.status = source["status"];
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

}

