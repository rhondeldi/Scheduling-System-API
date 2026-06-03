package Admin

type AdminCredentials struct {
	Username     string `json:"username" bson:"username"`
	PasswordHash string `json:"passwordHash" bson:"passwordHash"`
	Password     string `json:"password,omitempty" bson:"password,omitempty"`
}
