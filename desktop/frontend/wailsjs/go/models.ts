export namespace acl {
	
	export class Config {
	    BaseURL: string;
	    Timeout: number;
	    Proxy: string;
	    Step: number;
	    UseRSS: boolean;
	    UseBibTeX: boolean;
	
	    static createFrom(source: any = {}) {
	        return new Config(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.BaseURL = source["BaseURL"];
	        this.Timeout = source["Timeout"];
	        this.Proxy = source["Proxy"];
	        this.Step = source["Step"];
	        this.UseRSS = source["UseRSS"];
	        this.UseBibTeX = source["UseBibTeX"];
	    }
	}

}

export namespace arxiv {
	
	export class Config {
	    UseAPI: boolean;
	    Proxy: string;
	    Step: number;
	    Timeout: number;
	    APIBase: string;
	    WebBase: string;
	    NewBase: string;
	
	    static createFrom(source: any = {}) {
	        return new Config(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.UseAPI = source["UseAPI"];
	        this.Proxy = source["Proxy"];
	        this.Step = source["Step"];
	        this.Timeout = source["Timeout"];
	        this.APIBase = source["APIBase"];
	        this.WebBase = source["WebBase"];
	        this.NewBase = source["NewBase"];
	    }
	}

}

export namespace config {
	
	export class LLMConfig {
	    BaseURL: string;
	    ModelName: string;
	    APIKey: string;
	
	    static createFrom(source: any = {}) {
	        return new LLMConfig(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.BaseURL = source["BaseURL"];
	        this.ModelName = source["ModelName"];
	        this.APIKey = source["APIKey"];
	    }
	}
	export class DatabaseConfig {
	    Path: string;
	
	    static createFrom(source: any = {}) {
	        return new DatabaseConfig(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.Path = source["Path"];
	    }
	}
	export class AppConfig {
	    Env: string;
	    Embedder: embedding.EmbedderConfig;
	    Database: DatabaseConfig;
	    Zotero: core.ZoteroConfig;
	    FeiShu: core.FeiShuConfig;
	    Arxiv: arxiv.Config;
	    OpenReview: openreview.Config;
	    ACL: acl.Config;
	    SSRN: ssrn.Config;
	    LLM: LLMConfig;
	
	    static createFrom(source: any = {}) {
	        return new AppConfig(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.Env = source["Env"];
	        this.Embedder = this.convertValues(source["Embedder"], embedding.EmbedderConfig);
	        this.Database = this.convertValues(source["Database"], DatabaseConfig);
	        this.Zotero = this.convertValues(source["Zotero"], core.ZoteroConfig);
	        this.FeiShu = this.convertValues(source["FeiShu"], core.FeiShuConfig);
	        this.Arxiv = this.convertValues(source["Arxiv"], arxiv.Config);
	        this.OpenReview = this.convertValues(source["OpenReview"], openreview.Config);
	        this.ACL = this.convertValues(source["ACL"], acl.Config);
	        this.SSRN = this.convertValues(source["SSRN"], ssrn.Config);
	        this.LLM = this.convertValues(source["LLM"], LLMConfig);
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

export namespace core {
	
	export class FeiShuConfig {
	    AppID: string;
	    AppSecret: string;
	
	    static createFrom(source: any = {}) {
	        return new FeiShuConfig(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.AppID = source["AppID"];
	        this.AppSecret = source["AppSecret"];
	    }
	}
	export class ZoteroConfig {
	    UserID: string;
	    APIKey: string;
	
	    static createFrom(source: any = {}) {
	        return new ZoteroConfig(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.UserID = source["UserID"];
	        this.APIKey = source["APIKey"];
	    }
	}

}

export namespace embedding {
	
	export class EmbedderConfig {
	    BaseURL: string;
	    APIKey: string;
	    ModelName: string;
	    Dim: number;
	
	    static createFrom(source: any = {}) {
	        return new EmbedderConfig(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.BaseURL = source["BaseURL"];
	        this.APIKey = source["APIKey"];
	        this.ModelName = source["ModelName"];
	        this.Dim = source["Dim"];
	    }
	}

}

export namespace main {
	
	export class CleanOptions {
	    source: string;
	    from: string;
	    until: string;
	    withoutEmbed: boolean;
	    exportBefore: boolean;
	    exportFormat: string;
	    exportOutput: string;
	
	    static createFrom(source: any = {}) {
	        return new CleanOptions(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.source = source["source"];
	        this.from = source["from"];
	        this.until = source["until"];
	        this.withoutEmbed = source["withoutEmbed"];
	        this.exportBefore = source["exportBefore"];
	        this.exportFormat = source["exportFormat"];
	        this.exportOutput = source["exportOutput"];
	    }
	}
	export class CleanResult {
	    matched: number;
	    deleted: number;
	    exportedPath?: string;
	
	    static createFrom(source: any = {}) {
	        return new CleanResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.matched = source["matched"];
	        this.deleted = source["deleted"];
	        this.exportedPath = source["exportedPath"];
	    }
	}
	export class ExportOptions {
	    format: string;
	    output: string;
	    query: string;
	    keywords: string[];
	    categories: string[];
	    source: string;
	    collection: string;
	    feishuName: string;
	    limit: number;
	
	    static createFrom(source: any = {}) {
	        return new ExportOptions(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.format = source["format"];
	        this.output = source["output"];
	        this.query = source["query"];
	        this.keywords = source["keywords"];
	        this.categories = source["categories"];
	        this.source = source["source"];
	        this.collection = source["collection"];
	        this.feishuName = source["feishuName"];
	        this.limit = source["limit"];
	    }
	}
	export class PaperListResponse {
	    papers: models.Paper[];
	    total: number;
	
	    static createFrom(source: any = {}) {
	        return new PaperListResponse(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.papers = this.convertValues(source["papers"], models.Paper);
	        this.total = source["total"];
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
	export class RecommendOptions {
	    interestQuery: string;
	    platforms: string[];
	    zoteroCollection: string;
	    topK: number;
	    maxRecommendations: number;
	    forceCrawl: boolean;
	    dateFrom: string;
	    dateTo: string;
	    localFilePath: string;
	    localFileAction: string;
	
	    static createFrom(source: any = {}) {
	        return new RecommendOptions(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.interestQuery = source["interestQuery"];
	        this.platforms = source["platforms"];
	        this.zoteroCollection = source["zoteroCollection"];
	        this.topK = source["topK"];
	        this.maxRecommendations = source["maxRecommendations"];
	        this.forceCrawl = source["forceCrawl"];
	        this.dateFrom = source["dateFrom"];
	        this.dateTo = source["dateTo"];
	        this.localFilePath = source["localFilePath"];
	        this.localFileAction = source["localFileAction"];
	    }
	}
	export class SearchExample {
	    title: string;
	    abstract: string;
	
	    static createFrom(source: any = {}) {
	        return new SearchExample(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.title = source["title"];
	        this.abstract = source["abstract"];
	    }
	}
	export class SearchOptions {
	    query: string;
	    examples: SearchExample[];
	    semantic: boolean;
	    topK: number;
	    limit: number;
	    source: string;
	    from: string;
	    until: string;
	    computeEmbed: boolean;
	    embedBatch: number;
	
	    static createFrom(source: any = {}) {
	        return new SearchOptions(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.query = source["query"];
	        this.examples = this.convertValues(source["examples"], SearchExample);
	        this.semantic = source["semantic"];
	        this.topK = source["topK"];
	        this.limit = source["limit"];
	        this.source = source["source"];
	        this.from = source["from"];
	        this.until = source["until"];
	        this.computeEmbed = source["computeEmbed"];
	        this.embedBatch = source["embedBatch"];
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

export namespace models {
	
	export class Paper {
	    ID: number;
	    Source: string;
	    SourceID: string;
	    URL: string;
	    Title: string;
	    TitleTranslated: string;
	    Authors: string[];
	    Abstract: string;
	    AbstractTranslated: string;
	    Categories: string[];
	    Comments: string;
	    // Go type: time
	    FirstSubmittedAt: any;
	    // Go type: time
	    FirstAnnouncedAt: any;
	    // Go type: time
	    UpdatedAt: any;
	
	    static createFrom(source: any = {}) {
	        return new Paper(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.ID = source["ID"];
	        this.Source = source["Source"];
	        this.SourceID = source["SourceID"];
	        this.URL = source["URL"];
	        this.Title = source["Title"];
	        this.TitleTranslated = source["TitleTranslated"];
	        this.Authors = source["Authors"];
	        this.Abstract = source["Abstract"];
	        this.AbstractTranslated = source["AbstractTranslated"];
	        this.Categories = source["Categories"];
	        this.Comments = source["Comments"];
	        this.FirstSubmittedAt = this.convertValues(source["FirstSubmittedAt"], null);
	        this.FirstAnnouncedAt = this.convertValues(source["FirstAnnouncedAt"], null);
	        this.UpdatedAt = this.convertValues(source["UpdatedAt"], null);
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

export namespace openreview {
	
	export class Config {
	    APIBase: string;
	    Proxy: string;
	    Timeout: number;
	
	    static createFrom(source: any = {}) {
	        return new Config(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.APIBase = source["APIBase"];
	        this.Proxy = source["Proxy"];
	        this.Timeout = source["Timeout"];
	    }
	}

}

export namespace ssrn {
	
	export class Config {
	    Timeout: number;
	    Proxy: string;
	    BaseURL: string;
	    PageSize: number;
	    MaxPages: number;
	    RateLimitPerSecond: number;
	    Sort: string;
	
	    static createFrom(source: any = {}) {
	        return new Config(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.Timeout = source["Timeout"];
	        this.Proxy = source["Proxy"];
	        this.BaseURL = source["BaseURL"];
	        this.PageSize = source["PageSize"];
	        this.MaxPages = source["MaxPages"];
	        this.RateLimitPerSecond = source["RateLimitPerSecond"];
	        this.Sort = source["Sort"];
	    }
	}

}

