package zotero

import (
	"encoding/json"
)

// IntOrBool 处理可能是 int 或 false 的字段（如 numChildren）
// Zotero API 在值为 0 时会返回 false 而不是 0
type IntOrBool int

func (i *IntOrBool) UnmarshalJSON(data []byte) error {
	// 尝试解析为 bool
	var b bool
	if err := json.Unmarshal(data, &b); err == nil {
		if b {
			*i = 1
		} else {
			*i = 0
		}
		return nil
	}

	// 尝试解析为 int
	var num int
	if err := json.Unmarshal(data, &num); err == nil {
		*i = IntOrBool(num)
		return nil
	}

	// 默认为 0
	*i = 0
	return nil
}

func (i IntOrBool) MarshalJSON() ([]byte, error) {
	if i == 0 {
		return []byte("false"), nil
	}
	return json.Marshal(int(i))
}

func (i IntOrBool) Int() int {
	return int(i)
}

// StringOrBool 处理可能是 string 或 false 的字段（如 parentCollection）
type StringOrBool string

func (s *StringOrBool) UnmarshalJSON(data []byte) error {
	// 尝试解析为 bool
	var b bool
	if err := json.Unmarshal(data, &b); err == nil {
		*s = ""
		return nil
	}

	// 尝试解析为 string
	var str string
	if err := json.Unmarshal(data, &str); err == nil {
		*s = StringOrBool(str)
		return nil
	}

	*s = ""
	return nil
}

func (s StringOrBool) MarshalJSON() ([]byte, error) {
	if s == "" {
		return []byte("false"), nil
	}
	return json.Marshal(string(s))
}

func (s StringOrBool) String() string {
	return string(s)
}

// ==================== 核心数据结构 ====================

/*
	 	====================
		获取条目部分

GET /users/{userID}/items

		[
	  {
	    "key": "ABCD1234",
	    "version": 12345,
	    "library": {
	      "type": "user",
	      "id": 123456,
	      "name": "Username",
	      "links": {
	        "alternate": {
	          "href": "https://www.zotero.org/username",
	          "type": "text/html"
	        }
	      }
	    },
	    "links": {
	      "self": {
	        "href": "https://api.zotero.org/users/123456/items/ABCD1234",
	        "type": "application/json"
	      },
	      "alternate": {
	        "href": "https://www.zotero.org/username/items/ABCD1234",
	        "type": "text/html"
	      }
	    },
	    "meta": {
	      "createdByUser": {
	        "id": 123456,
	        "username": "username",
	        "name": "User Name",
	        "links": {}
	      },
	      "creatorSummary": "Smith et al.",
	      "parsedDate": "2024-01-15",
	      "numChildren": 1  // 注意：可能是 false 或数字
	    },
	    "data": {
	      "key": "ABCD1234",
	      "version": 12345,
	      "itemType": "journalArticle",  // 或 "preprint", "conferencePaper" 等
	      "title": "Deep Learning for Natural Language Processing",
	      "creators": [
	        {
	          "creatorType": "author",
	          "firstName": "John",
	          "lastName": "Smith"
	        }
	      ],
	      "abstractNote": "This paper presents a novel approach...",
	      "publicationTitle": "Nature",
	      "date": "2024-01-15",
	      "DOI": "10.1038/s41586-024-12345-6",
	      "url": "https://arxiv.org/abs/2401.12345",
	      "extra": "arXiv:2401.12345",
	      "tags": [
	        {
	          "tag": "machine learning",
	          "type": 1
	        }
	      ],
	      "collections": ["EFGH5678"],
	      "relations": {},
	      "dateAdded": "2024-01-15T10:00:00Z",
	      "dateModified": "2024-01-15T12:00:00Z"
	    }
	  }

]

	====================
*/
type Item struct {
	Key     string   `json:"key"`
	Version int      `json:"version"`
	Library Library  `json:"library"`
	Links   Links    `json:"links"`
	Meta    ItemMeta `json:"meta"`
	Data    ItemData `json:"data"`
}

// ItemData 条目的核心数据
// 注意：不同 itemType 有不同的可用字段
type ItemData struct {
	Key     *string `json:"key,omitempty"`
	Version *int    `json:"version,omitempty"`

	// ===== 基础字段（所有类型都有） =====
	ItemType     string    `json:"itemType"` // journalArticle, preprint, conferencePaper, book 等
	Title        string    `json:"title"`
	Creators     []Creator `json:"creators,omitempty"`
	AbstractNote *string   `json:"abstractNote,omitempty"`
	Date         *string   `json:"date,omitempty"`
	DateAdded    string    `json:"dateAdded,omitempty"`
	DateModified string    `json:"dateModified,omitempty"`

	// ===== 标识符 =====
	DOI   *string `json:"DOI,omitempty"`
	ISBN  *string `json:"ISBN,omitempty"`
	ISSN  *string `json:"ISSN,omitempty"`
	ArXiv *string `json:"arXiv,omitempty"` // arXiv ID（如果作为独立字段）

	// ===== URL 和访问 =====
	URL        *string `json:"url,omitempty"`
	AccessDate *string `json:"accessDate,omitempty"`

	// ===== 出版信息 =====
	PublicationTitle    *string `json:"publicationTitle,omitempty"` // 期刊/会议名
	JournalAbbreviation *string `json:"journalAbbreviation,omitempty"`
	Volume              *string `json:"volume,omitempty"`
	Issue               *string `json:"issue,omitempty"`
	Pages               *string `json:"pages,omitempty"`
	Series              *string `json:"series,omitempty"`
	SeriesTitle         *string `json:"seriesTitle,omitempty"`
	SeriesText          *string `json:"seriesText,omitempty"`
	ConferenceName      *string `json:"conferenceName,omitempty"` // 会议专用
	Place               *string `json:"place,omitempty"`          // 出版地
	Publisher           *string `json:"publisher,omitempty"`

	// ===== 其他元数据 =====
	Language        *string `json:"language,omitempty"`
	ShortTitle      *string `json:"shortTitle,omitempty"`
	Rights          *string `json:"rights,omitempty"`
	Extra           *string `json:"extra,omitempty"` // 额外信息（常用于存 arXiv ID）
	Archive         *string `json:"archive,omitempty"`
	ArchiveLocation *string `json:"archiveLocation,omitempty"`
	LibraryCatalog  *string `json:"libraryCatalog,omitempty"`
	CallNumber      *string `json:"callNumber,omitempty"`

	// ===== 组织信息 =====
	Tags        []Tag                  `json:"tags,omitempty"`
	Collections []string               `json:"collections,omitempty"` // Collection keys
	Relations   map[string]interface{} `json:"relations,omitempty"`

	// ===== 预印本专用字段 =====
	Repository  *string `json:"repository,omitempty"`  // 预印本仓库（如 arXiv）
	ArchiveID   *string `json:"archiveID,omitempty"`   // 预印本 ID
	CitationKey *string `json:"citationKey,omitempty"` // BibTeX 引用键

	// ===== 书籍专用字段 =====
	Edition  *string `json:"edition,omitempty"`
	NumPages *string `json:"numPages,omitempty"`
}

// Creator 作者/编辑等创作者信息
type Creator struct {
	CreatorType string `json:"creatorType"` // author, editor, contributor, translator 等
	FirstName   string `json:"firstName,omitempty"`
	LastName    string `json:"lastName,omitempty"`
	Name        string `json:"name,omitempty"` // 单一名称（用于机构等）
}

// Tag 标签
type Tag struct {
	Tag  string `json:"tag"`
	Type int    `json:"type,omitempty"` // 0=用户标签, 1=自动标签
}

// ItemMeta 条目元数据（由 Zotero 生成）
type ItemMeta struct {
	CreatedByUser  *User     `json:"createdByUser,omitempty"`
	CreatorSummary string    `json:"creatorSummary,omitempty"`
	ParsedDate     string    `json:"parsedDate,omitempty"`
	NumChildren    IntOrBool `json:"numChildren"` // 注意：可能是 false 或数字
}

// User 用户信息
type User struct {
	ID       int    `json:"id"`
	Username string `json:"username"`
	Name     string `json:"name"`
	Links    Links  `json:"links"`
}

// Library 库信息
type Library struct {
	Type  string `json:"type"` // "user" or "group"
	ID    int    `json:"id"`
	Name  string `json:"name"`
	Links Links  `json:"links"`
}

// Links 链接集合
type Links struct {
	Self      *Link `json:"self,omitempty"`
	Alternate *Link `json:"alternate,omitempty"`
	Up        *Link `json:"up,omitempty"`
}

// Link 单个链接
type Link struct {
	Href string `json:"href"`
	Type string `json:"type"`
}

// Meta 通用元数据
type Meta struct {
	Created      string    `json:"created"`
	LastModified string    `json:"lastModified"`
	NumItems     IntOrBool `json:"numItems,omitempty"`
}

/*
	 	====================
		创建条目部分

POST /users/{userID}/items

[

	{
	  "itemType": "preprint",
	  "title": "Attention Is All You Need",
	  "creators": [
	    {
	      "creatorType": "author",
	      "firstName": "Ashish",
	      "lastName": "Vaswani"
	    }
	  ],
	  "abstractNote": "The dominant sequence transduction models...",
	  "date": "2017-06-12",
	  "url": "https://arxiv.org/abs/1706.03762",
	  "extra": "arXiv:1706.03762",
	  "tags": [
	    {"tag": "transformer"},
	    {"tag": "attention"}
	  ],
	  "collections": ["EFGH5678"]
	}

]

响应：

	{
	  "successful": {
	    "0": {
	      "key": "NEWKEY123",
	      "version": 1,
	      "library": { /* ... * },
	      "links": { /* ... * },
	      "meta": { /* ... * },
	      "data": { /* 完整的 ItemData * }
	    }
	  },
	  "unchanged": {},
	  "failed": {}
	}

常见响应
200 OK 	请求已完成。请查看响应 JSON 以获取单个写入的状态。
400 Bad Request 	无效的类型/字段；无法解析的 JSON
409 Conflict 	目标库已被锁定。
412 Precondition Failed 	在 If-Unmodified-Since-Version 中提供的版本已过时，或者已提交提供的 Zotero-Write-Token。
413 Request Entity Too Large 	提交项目过多
*/
type CreateResponse struct {
	Successful map[string]Item       `json:"successful"`
	Unchanged  map[string]Item       `json:"unchanged"`
	Failed     map[string]FailedItem `json:"failed"`
}

type FailedItem struct {
	Key     string `json:"key"`
	Code    int    `json:"code"`
	Message string `json:"message"`
}

// Collection 集合
type Collection struct {
	Key     string         `json:"key"`
	Version int            `json:"version"`
	Library Library        `json:"library"`
	Links   Links          `json:"links"`
	Meta    CollectionMeta `json:"meta"`
	Data    CollectionData `json:"data"`
}

// CollectionData 集合数据
type CollectionData struct {
	Key              string                 `json:"key"`
	Version          int                    `json:"version"`
	Name             string                 `json:"name"`
	ParentCollection StringOrBool           `json:"parentCollection"` // false 或父集合 key
	Relations        map[string]interface{} `json:"relations"`
}

// CollectionMeta 集合元数据
type CollectionMeta struct {
	NumCollections IntOrBool `json:"numCollections"`
	NumItems       IntOrBool `json:"numItems"`
}
