package domain

type User struct {
	Id       int    `json:"id"`
	Fname    string `json:"fname"`
	Lname    string `json:"lname"`
	Username string `json:"username"`
	Email    string `json:"email"`
	Password string `json:"-"`
}
