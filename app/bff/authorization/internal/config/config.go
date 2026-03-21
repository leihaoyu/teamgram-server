// Copyright 2022 Teamgram Authors
//  All rights reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//   http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
//
// Author: teamgramio (teamgram.io@gmail.com)
//

package config

import (
	kafka "github.com/teamgram/marmota/pkg/mq"
	"github.com/teamgram/marmota/pkg/stores/sqlx"
	"github.com/teamgram/teamgram-server/pkg/code/conf"
	"github.com/zeromicro/go-zero/core/stores/kv"
	"github.com/zeromicro/go-zero/zrpc"
)

// AuthMethodUsernamePassword 用户名+密码
const AuthMethodUsernamePassword = "username_password"

// AuthMethodPhoneSmsCode 手机号+验证码
const AuthMethodPhoneSmsCode = "phone_sms_code"

// AuthMethodPhonePassword 手机号+密码
const AuthMethodPhonePassword = "phone_password"

type Config struct {
	zrpc.RpcServerConf
	AuthMethods               []string `json:",optional"` // 认证方式列表，如 ["username_password", "phone_sms_code"]
	KV                        kv.KvConf
	Code                      *conf.SmsVerifyCodeConfig
	UserClient                zrpc.RpcClientConf
	AuthsessionClient         zrpc.RpcClientConf
	ChatClient                zrpc.RpcClientConf
	StatusClient              zrpc.RpcClientConf
	UsernameClient            zrpc.RpcClientConf
	MsgClient                 zrpc.RpcClientConf
	SyncClient                *kafka.KafkaProducerConf
	SignInServiceNotification []conf.MessageEntityConfig `json:",optional"`
	SignInMessage             []conf.MessageEntityConfig `json:",optional"`
	AutoGroupMySQL            *sqlx.Config               `json:",optional"`
	SystemAdminUserId         int64                      `json:",default=777001"`
}

// GetAuthMethods 获取配置的认证方式列表，默认返回 ["username_password", "phone_password"]
func (c *Config) GetAuthMethods() []string {
	if len(c.AuthMethods) == 0 {
		return []string{AuthMethodUsernamePassword, AuthMethodPhonePassword}
	}
	return c.AuthMethods
}
