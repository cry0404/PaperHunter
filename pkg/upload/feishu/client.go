package feishu

import (
	"bytes"
	"context"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"net/http"
	"os"

	lark "github.com/larksuite/oapi-sdk-go/v3"
	larkcore "github.com/larksuite/oapi-sdk-go/v3/core"
	larkbitable "github.com/larksuite/oapi-sdk-go/v3/service/bitable/v1"
	//larkdrive "github.com/larksuite/oapi-sdk-go/v3/service/drive/v1"
)

//const rootFolderEndpoint = "https://open.feishu.cn/open-apis/drive/explorer/v2/root_folder/meta"

// 上传 file 的时候可以使用 wails 的 runtime 来管理？

// 可以指定名字，选取对应的文件，所以 feishu 的 config 需要 appid
type Client struct {
	AppID        string
	AppSecret    string
	FileName     string
	FolderName   string
	httpClient   *http.Client
	feishuClient *lark.Client
}

// NewClient 创建新的飞书客户端
func NewClient(appID, appSecret, fileName, folderName string) *Client {
	return &Client{
		AppID:        appID,
		AppSecret:    appSecret,
		FileName:     fileName,
		FolderName:   folderName, //飞书上多维表格的名字
		httpClient:   http.DefaultClient,
		feishuClient: lark.NewClient(appID, appSecret),
	}
}

// getTenantAccessToken 获取 Tenant Access Token
func (c *Client) getTenantAccessToken() (string, error) {
	url := "https://open.feishu.cn/open-apis/auth/v3/tenant_access_token/internal"

	data := map[string]string{
		"app_id":     c.AppID,
		"app_secret": c.AppSecret,
	}

	jsonData, err := json.Marshal(data)
	if err != nil {
		return "", fmt.Errorf("marshal data error: %w", err)
	}

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return "", fmt.Errorf("create request error: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("send request error: %w", err)
	}
	defer resp.Body.Close()

	var result struct {
		Code int    `json:"code"`
		Msg  string `json:"msg"`
		Data struct {
			TenantAccessToken string `json:"tenant_access_token"`
			Expire            int    `json:"expire"`
		} `json:"data"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("decode response error: %w", err)
	}

	if result.Code != 0 {
		return "", fmt.Errorf("API error: code=%d, msg=%s", result.Code, result.Msg)
	}

	return result.Data.TenantAccessToken, nil
}

/*
// getRootFolderToken 获取根文件夹 Token
func (c *Client) getRootFolderToken(tenantAccessToken string) (string, error) {
	req, err := http.NewRequest("GET", rootFolderEndpoint, nil)
	if err != nil {
		return "", fmt.Errorf("create request error: %w", err)
	}

	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", tenantAccessToken))
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("send request error: %w", err)
	}
	defer resp.Body.Close()

	// 读取响应体以便调试
	body, _ := io.ReadAll(resp.Body)

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("HTTP error: %d, body: %s", resp.StatusCode, string(body))
	}

	var result Response
	if err := json.Unmarshal(body, &result); err != nil {
		return "", fmt.Errorf("decode response error: %w, body: %s", err, string(body))
	}

	if result.Code != 0 {
		return "", fmt.Errorf("API error: code=%d, msg=%s", result.Code, result.Msg)
	}

	return result.Data.Token, nil
}*/
/*
// createFolder 创建文件夹
func (c *Client) createFolder(folderName, parentFolderToken, tenantAccessToken string) (string, error) {
	req := larkdrive.NewCreateFolderFileReqBuilder().
		Body(larkdrive.NewCreateFolderFileReqBodyBuilder().
			Name(folderName).
			FolderToken(parentFolderToken).
			Build()).
		Build()

	resp, err := c.feishuClient.Drive.V1.File.CreateFolder(
		context.Background(),
		req,
		larkcore.WithTenantAccessToken(tenantAccessToken),
	)

	if err != nil {
		return "", fmt.Errorf("create folder error: %w", err)
	}

	if !resp.Success() {
		return "", fmt.Errorf("create folder failed: logId=%s, error=%s",
			resp.RequestId(), larkcore.Prettify(resp.CodeError))
	}

	// 从响应中获取创建的文件夹 token
	token := resp.Data.Token
	return *token, nil
}
*/

// createBitable 创建多维表格
func (c *Client) createBitable(fileName, parentFolderToken, tenantAccessToken string) (string, string, error) {
	req := larkbitable.NewCreateAppReqBuilder().
		ReqApp(larkbitable.NewReqAppBuilder().
			Name(fileName).
			FolderToken(parentFolderToken).
			Build()).
		Build()

	// 发起请求
	resp, err := c.feishuClient.Bitable.V1.App.Create(context.Background(), req, larkcore.WithTenantAccessToken(tenantAccessToken))

	if err != nil {
		return "", "", fmt.Errorf("create bitable error: %w", err)
	}

	if !resp.Success() {
		return "", "", fmt.Errorf("create bitable failed: logId=%s, error=%s",
			resp.RequestId(), larkcore.Prettify(resp.CodeError))
	}

	appToken := resp.Data.App.AppToken
	url := ""
	if resp.Data.App.Url != nil {
		url = *resp.Data.App.Url
		//fmt.Println("当前多维表格的 url 是，请妥善存储（我目前没有找到别的能获取自建应用的云文档 url 的方法): ", url)
	}
	return *appToken, url, nil
}

// createTableInBitable 在多维表格中创建数据表
func (c *Client) createTableInBitable(appToken, tableName string, fields []*larkbitable.AppTableCreateHeader, tenantAccessToken string) (string, error) {
	req := larkbitable.NewCreateAppTableReqBuilder().
		AppToken(appToken).
		Body(larkbitable.NewCreateAppTableReqBodyBuilder().
			Table(larkbitable.NewReqTableBuilder().
				Name(tableName).
				DefaultViewName("默认视图").
				Fields(fields).
				Build()).
			Build()).
		Build()

	// 发起请求
	resp, err := c.feishuClient.Bitable.V1.AppTable.Create(context.Background(), req, larkcore.WithTenantAccessToken(tenantAccessToken))

	if err != nil {
		return "", fmt.Errorf("create table error: %w", err)
	}

	if !resp.Success() {
		return "", fmt.Errorf("create table failed: logId=%s, error=%s",
			resp.RequestId(), larkcore.Prettify(resp.CodeError))
	}

	if resp.Data.TableId == nil {
		return "", fmt.Errorf("tableId is nil")
	}

	return *resp.Data.TableId, nil
}

// addRecordsToBitable 向多维表格添加记录
func (c *Client) addRecordsToBitable(appToken, tableId string, records []*larkbitable.AppTableRecord, tenantAccessToken string) error {
	batchSize := 1000

	for i := 0; i < len(records); i += batchSize {
		end := i + batchSize
		if end > len(records) {
			end = len(records)
		}

		batch := records[i:end]

		req := larkbitable.NewBatchCreateAppTableRecordReqBuilder().
			AppToken(appToken).
			TableId(tableId).
			Body(larkbitable.NewBatchCreateAppTableRecordReqBodyBuilder().
				Records(batch).
				Build()).
			Build()

		// 发起请求
		resp, err := c.feishuClient.Bitable.V1.AppTableRecord.BatchCreate(context.Background(), req, larkcore.WithTenantAccessToken(tenantAccessToken))

		if err != nil {
			return fmt.Errorf("add records error: %w", err)
		}

		if !resp.Success() {
			return fmt.Errorf("add records failed: logId=%s, error=%s",
				resp.RequestId(), larkcore.Prettify(resp.CodeError))
		}

	}

	return nil
}

// parseCSVFile 解析 CSV 文件并返回表头和记录
func (c *Client) parseCSVFile(filePath string) ([]string, [][]string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, nil, fmt.Errorf("打开文件失败: %w", err)
	}
	defer file.Close()

	reader := csv.NewReader(file)

	// 读取表头
	headers, err := reader.Read()
	if err != nil {
		return nil, nil, fmt.Errorf("读取表头失败: %w", err)
	}

	// 读取所有记录
	records, err := reader.ReadAll()
	if err != nil {
		return nil, nil, fmt.Errorf("读取记录失败: %w", err)
	}

	return headers, records, nil
}

// convertCSVToBitableRecords 将 CSV 记录转换为飞书记录格式
func (c *Client) convertCSVToBitableRecords(headers []string, csvRecords [][]string) ([]*larkbitable.AppTableRecord, error) {
	records := make([]*larkbitable.AppTableRecord, len(csvRecords))

	for i, csvRow := range csvRecords {
		// 创建字段映射
		fields := make(map[string]interface{})

		for j, header := range headers {
			if j < len(csvRow) {
				// 字段值为字符串
				fields[header] = csvRow[j]
			}
		}

		records[i] = larkbitable.NewAppTableRecordBuilder().
			Fields(fields).
			Build()
	}

	return records, nil
}

// UploadCSVToBitable 将 CSV 上传为多维表格
func (c *Client) UploadCSVToBitable(csvFilePath string) (string, error) {

	headers, records, err := c.parseCSVFile(csvFilePath)
	if err != nil {
		return "", fmt.Errorf("解析 CSV 失败: %w", err)
	}

	fmt.Printf("CSV 文件包含 %d 列，%d 行数据\n", len(headers), len(records))

	tenantAccessToken, err := c.getTenantAccessToken()
	if err != nil {
		return "", fmt.Errorf("获取 tenant access token 失败: %w", err)
	}

	//这里应用的多维表格似乎申请不了权限
	bitableToken, bitableURL, err := c.createBitable(c.FileName, "", tenantAccessToken)
	if err != nil {
		return "", fmt.Errorf("创建多维表格失败: %w", err)
	}
	//fmt.Printf("创建多维表格成功，token: %s\n", bitableToken)

	fields := make([]*larkbitable.AppTableCreateHeader, len(headers)+1)
	for i, header := range headers {
		fields[i] = larkbitable.NewAppTableCreateHeaderBuilder().
			FieldName(header).
			Type(1).
			Build()
	}

	fields[len(headers)] = larkbitable.NewAppTableCreateHeaderBuilder().
		FieldName("评价"). //留给 ai 做总结字段
		Type(1).
		Build()

	tableId, err := c.createTableInBitable(bitableToken, "数据集", fields, tenantAccessToken)
	if err != nil {
		return "", fmt.Errorf("创建数据表失败: %w", err)
	}

	bitableRecords, err := c.convertCSVToBitableRecords(headers, records)
	if err != nil {
		return "", fmt.Errorf("转换记录失败: %w", err)
	}

	if err := c.addRecordsToBitable(bitableToken, tableId, bitableRecords, tenantAccessToken); err != nil {
		return "", fmt.Errorf("添加记录失败: %w", err)
	}

	return bitableURL, nil
}
