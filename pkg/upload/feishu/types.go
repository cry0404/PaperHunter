package feishu


type Data struct {
	Token  string `json:"token"`
	ID     string `json:"id"`
	UserID string `json:"user_id"`
}

type Response struct {
	Code int    `json:"code"`
	Msg  string `json:"msg"`
	Data Data   `json:"data"`
}