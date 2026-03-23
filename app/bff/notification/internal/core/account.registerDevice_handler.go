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

package core

import (
	"encoding/hex"

	"github.com/teamgram/proto/mtproto"
	"github.com/teamgram/teamgram-server/app/bff/notification/internal/dao"
)

// AccountRegisterDevice
// account.registerDevice#ec86017a flags:# no_muted:flags.0?true token_type:int token:string app_sandbox:Bool secret:bytes other_uids:Vector<long> = Bool;
func (c *NotificationCore) AccountRegisterDevice(in *mtproto.TLAccountRegisterDevice) (*mtproto.Bool, error) {
	c.Logger.Infof("account.registerDevice - userId: %d, tokenType: %d, token: %s, permAuthKeyId: %d",
		c.MD.UserId, in.TokenType, in.Token, c.MD.PermAuthKeyId)

	err := c.svcCtx.Dao.RegisterDevice(c.ctx, &dao.DevicesDO{
		AuthKeyId:  c.MD.PermAuthKeyId,
		UserId:     c.MD.UserId,
		TokenType:  in.TokenType,
		Token:      in.Token,
		NoMuted:    in.GetNoMuted(),
		AppSandbox: mtproto.FromBool(in.AppSandbox),
		Secret:     hex.EncodeToString(in.Secret),
		OtherUids:  dao.JoinInt64s(in.OtherUids),
	})
	if err != nil {
		c.Logger.Errorf("account.registerDevice - error: %v", err)
		return nil, err
	}

	return mtproto.BoolTrue, nil
}
