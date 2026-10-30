package security

import (
	"testing"

	"github.com/fino-io/nacos-sdk-go/v2/common/constant"
	"github.com/stretchr/testify/assert"
)

func TestStaticRamCredentials(t *testing.T) {
	for _, token := range []string{"", "sts-token"} {
		t.Run(token, func(t *testing.T) {
			t.Setenv(ACCESS_KEY_ID_KEY, "")
			t.Setenv(ACCESS_KEY_SECRET_KEY, "")
			t.Setenv(SECURITY_TOKEN_KEY, "")
			client := NewRamAuthClient(constant.ClientConfig{
				AccessKey: "ak", SecretKey: "sk",
				RamConfig: &constant.RamConfig{SecurityToken: token, SignatureRegionId: "region"},
			})
			ok, err := client.Login()
			assert.NoError(t, err)
			assert.True(t, ok)
			assert.Equal(t, RamContext{
				AccessKey: "ak", SecretKey: "sk", SecurityToken: token,
				SignatureRegionId: "region", EphemeralAccessKeyId: token != "",
			}, client.matchedProvider.GetCredentialsForNacosClient())
		})
	}
}
