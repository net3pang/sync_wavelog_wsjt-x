export namespace main {
	
	export class ConfigDTO {
	    wavelog_url: string;
	    api_key: string;
	    station_id: string;
	    wsjtx_log_path: string;
	
	    static createFrom(source: any = {}) {
	        return new ConfigDTO(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.wavelog_url = source["wavelog_url"];
	        this.api_key = source["api_key"];
	        this.station_id = source["station_id"];
	        this.wsjtx_log_path = source["wsjtx_log_path"];
	    }
	}

}

export namespace wavelog {
	
	export class StationProfile {
	    station_id: string;
	    station_profile_name: string;
	
	    static createFrom(source: any = {}) {
	        return new StationProfile(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.station_id = source["station_id"];
	        this.station_profile_name = source["station_profile_name"];
	    }
	}

}

