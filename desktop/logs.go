package main

import (
	"fmt"
	"os"
)


func (a *App) GetLogs() (string, error) {
	if a.logfile == "" {
		return "", fmt.Errorf("日志文件未初始化")
	}

	content, err := os.ReadFile(a.logfile)
	if err != nil {
		return "", err
	}

	return string(content), nil
}

func (a *App) ClearLogs() error {
    if a.logfile == "" {
        return fmt.Errorf("日志文件未初始化")
    }
    
    return os.WriteFile(a.logfile, []byte{}, 0644)
}



