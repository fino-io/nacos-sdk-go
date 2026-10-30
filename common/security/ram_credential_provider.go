package security

import (
	"os"

	"github.com/fino-io/nacos-sdk-go/v2/common/constant"
)

const (
	ENV_PREFIX              string = "ALIBABA_CLOUD_"
	ACCESS_KEY_ID_KEY       string = ENV_PREFIX + "ACCESS_KEY_ID"
	ACCESS_KEY_SECRET_KEY   string = ENV_PREFIX + "ACCESS_KEY_SECRET"
	SECURITY_TOKEN_KEY      string = ENV_PREFIX + "SECURITY_TOKEN"
	SIGNATURE_REGION_ID_KEY string = ENV_PREFIX + "SIGNATURE_REGION_ID"
)

func GetNacosProperties(property string, envKey string) string {
	if property != "" {
		return property
	} else {
		return os.Getenv(envKey)
	}
}

type RamCredentialProvider interface {
	MatchProvider() bool
	Init() error
	GetCredentialsForNacosClient() RamContext
}

type AccessKeyCredentialProvider struct {
	clientConfig      constant.ClientConfig
	accessKey         string
	secretKey         string
	signatureRegionId string
}

func (provider *AccessKeyCredentialProvider) MatchProvider() bool {
	accessKey := GetNacosProperties(provider.clientConfig.AccessKey, ACCESS_KEY_ID_KEY)
	secretKey := GetNacosProperties(provider.clientConfig.SecretKey, ACCESS_KEY_SECRET_KEY)
	return accessKey != "" && secretKey != ""
}

func (provider *AccessKeyCredentialProvider) Init() error {
	provider.accessKey = GetNacosProperties(provider.clientConfig.AccessKey, ACCESS_KEY_ID_KEY)
	provider.secretKey = GetNacosProperties(provider.clientConfig.SecretKey, ACCESS_KEY_SECRET_KEY)
	if provider.clientConfig.RamConfig != nil {
		provider.signatureRegionId = GetNacosProperties(provider.clientConfig.RamConfig.SignatureRegionId, SIGNATURE_REGION_ID_KEY)
	} else {
		provider.signatureRegionId = ""
	}
	return nil
}

func (provider *AccessKeyCredentialProvider) GetCredentialsForNacosClient() RamContext {
	ramContext := RamContext{
		AccessKey:         provider.accessKey,
		SecretKey:         provider.secretKey,
		SignatureRegionId: provider.signatureRegionId,
	}
	return ramContext
}

type StsTokenCredentialProvider struct {
	clientConfig      constant.ClientConfig
	accessKey         string
	secretKey         string
	securityToken     string
	signatureRegionId string
}

func (provider *StsTokenCredentialProvider) MatchProvider() bool {
	accessKey := GetNacosProperties(provider.clientConfig.AccessKey, ACCESS_KEY_ID_KEY)
	secretKey := GetNacosProperties(provider.clientConfig.SecretKey, ACCESS_KEY_SECRET_KEY)
	if provider.clientConfig.RamConfig == nil {
		return false
	}
	stsToken := GetNacosProperties(provider.clientConfig.RamConfig.SecurityToken, SECURITY_TOKEN_KEY)
	return accessKey != "" && secretKey != "" && stsToken != ""
}

func (provider *StsTokenCredentialProvider) Init() error {
	provider.accessKey = GetNacosProperties(provider.clientConfig.AccessKey, ACCESS_KEY_ID_KEY)
	provider.secretKey = GetNacosProperties(provider.clientConfig.SecretKey, ACCESS_KEY_SECRET_KEY)
	provider.securityToken = GetNacosProperties(provider.clientConfig.RamConfig.SecurityToken, SECURITY_TOKEN_KEY)
	provider.signatureRegionId = GetNacosProperties(provider.clientConfig.RamConfig.SignatureRegionId, SIGNATURE_REGION_ID_KEY)
	return nil
}

func (provider *StsTokenCredentialProvider) GetCredentialsForNacosClient() RamContext {
	ramContext := RamContext{
		AccessKey:            provider.accessKey,
		SecretKey:            provider.secretKey,
		SecurityToken:        provider.securityToken,
		SignatureRegionId:    provider.signatureRegionId,
		EphemeralAccessKeyId: true,
	}
	return ramContext
}
