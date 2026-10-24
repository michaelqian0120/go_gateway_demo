package public

type AdminSessionInfo struct {
	Id        int64  `json:"id"`
	UserName  string `json:"user_name"`
	LoginTime int64  `json:"login_time"`
}
