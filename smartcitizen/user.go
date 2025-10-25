package smartcitizen

type Location struct {
	City        string `json:"city"`
	Country     string `json:"country"`
	CountryCode string `json:"country_code"`
}

type User struct {
	ID       int    `json:"id"`
	UUID     string `json:"uuid"`
	Username string `json:"username"`
	Email    string `json:"email"`
	Role     string `json:"role"`
	URL      string `json:"url"`

	Location Location     `json:"location"`
	Devices  []UserDevice `json:"devices"`
}
