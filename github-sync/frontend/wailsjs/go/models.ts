export namespace main {
	
	export class CloneResult {
	    success: boolean;
	    local_path: string;
	    remote_url: string;
	    error: string;
	
	    static createFrom(source: any = {}) {
	        return new CloneResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.success = source["success"];
	        this.local_path = source["local_path"];
	        this.remote_url = source["remote_url"];
	        this.error = source["error"];
	    }
	}
	export class Config {
	    token: string;
	    username: string;
	    storage_path: string;
	
	    static createFrom(source: any = {}) {
	        return new Config(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.token = source["token"];
	        this.username = source["username"];
	        this.storage_path = source["storage_path"];
	    }
	}
	export class FileStatus {
	    path: string;
	    status: string;
	    staged: boolean;
	    modified: boolean;
	
	    static createFrom(source: any = {}) {
	        return new FileStatus(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.path = source["path"];
	        this.status = source["status"];
	        this.staged = source["staged"];
	        this.modified = source["modified"];
	    }
	}
	export class LogEntry {
	    sha: string;
	    message: string;
	    author: string;
	    timestamp: string;
	
	    static createFrom(source: any = {}) {
	        return new LogEntry(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.sha = source["sha"];
	        this.message = source["message"];
	        this.author = source["author"];
	        this.timestamp = source["timestamp"];
	    }
	}
	export class RepoInfo {
	    name: string;
	    local_path: string;
	    remote_url: string;
	    last_sync_time: string;
	    branch: string;
	    commit_sha: string;
	
	    static createFrom(source: any = {}) {
	        return new RepoInfo(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.local_path = source["local_path"];
	        this.remote_url = source["remote_url"];
	        this.last_sync_time = source["last_sync_time"];
	        this.branch = source["branch"];
	        this.commit_sha = source["commit_sha"];
	    }
	}
	export class StatusResult {
	    clean: boolean;
	    files: FileStatus[];
	
	    static createFrom(source: any = {}) {
	        return new StatusResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.clean = source["clean"];
	        this.files = this.convertValues(source["files"], FileStatus);
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

