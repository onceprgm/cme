package account

import (
	"crypto/md5"
	"fmt"
)

type Account struct {
	Username    string
	UUID        string
	AccessToken string
	UserType    string
}

func Offline(username string) Account {
	return Account{
		Username:    username,
		UUID:        offlineUUID(username),
		AccessToken: "0",
		UserType:    "legacy",
	}
}

func offlineUUID(username string) string {
	sum := md5.Sum([]byte("OfflinePlayer:" + username))

	sum[6] = (sum[6] & 0x0f) | 0x30
	sum[8] = (sum[8] & 0x3f) | 0x80

	return fmt.Sprintf("%x-%x-%x-%x-%x",
		sum[0:4], sum[4:6], sum[6:8], sum[8:10], sum[10:16])
}
