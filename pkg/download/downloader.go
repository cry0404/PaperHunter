package download

import (
	"fmt"
	"sync"
	"time"
)

// TODO: 实现一个通用的 downloader，可以对应不同平台的下载
type Downloader struct {
	

}
var (
	downloader   *Downloader
	downloadOnce sync.Once
	path = "~/.quicksearch/pdfs"  //固定下载的存储点，但是需要考虑是否要定时删除
)


// proxy 预加载？
func getdownloader() *Downloader{
	downloadOnce.Do(func(){
		downloader = &Downloader{}
	})
	return downloader
}

// downloader 也应该是全局单例才对
func DownloadPDF(platform string, urls []string) ([]string, error) { //这里应该是把摘要位置放过去
	var pdfs []string
	var err error

	// 先只支持 arxiv
	if platform != "arxiv" {
		return nil, fmt.Errorf("暂不支持除 arxiv 外的平台")
	}
	switch platform {
	case "arxiv":
		pdfs, err = downloadArxivPDFs(urls)
		if err != nil {
			return nil, fmt.Errorf("下载 arxiv pdfs 发生错误")
		}
	case "openreview":
		pdfs, err = downloadOpenReviewPDFs(urls)
		if err != nil {
			return nil, fmt.Errorf("下载 openreview pdfs 发生错误")
		}
	case "acl":
		pdfs, err = downloadACLPDFs(urls)
		if err != nil {
			return nil, fmt.Errorf("下载 acl pdfs 发生错误")
		}
	case "ssrn":
		pdfs, err = downloadSSRNPDFs(urls)
		if err != nil {
			return nil, fmt.Errorf("下载 ssrn pdfs 发生错误")
		}
	default:
		return nil, fmt.Errorf("未知的平台")
	}

	return pdfs, nil
}


func downloadArxivPDFs(urls []string) ([]string, error) {


	return nil, nil
}

func downloadOpenReviewPDFs(urls []string) ([]string, error) {
	// TODO: 实现
	return nil, nil
}

func downloadACLPDFs(urls []string) ([]string, error) {
	// TODO: 实现
	return nil, nil
}

func downloadSSRNPDFs(urls []string) ([]string, error) {
	// TODO: 实现
	return nil, nil
}

func ClearPDFs(platform string) error { //应该在每个下载的 pdf 后加上时间后缀 
	return nil
}

func timeFormat(pdfName string) string {
	now := time.Now()

	dateStr := now.Format("2006-01-02")

	return pdfName + "_" + dateStr
}