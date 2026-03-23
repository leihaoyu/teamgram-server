// Copyright 2024 Teamgram Authors
//  All rights reserved.
//
// Author: teamgramio (teamgram.io@gmail.com)
//

package core

import (
	"github.com/teamgram/proto/mtproto"
	"github.com/teamgram/teamgram-server/app/service/authsession/authsession"
)

// AccountGetAuthorizations
// account.getAuthorizations#e320c158 = account.Authorizations;
func (c *AccountCore) AccountGetAuthorizations(in *mtproto.TLAccountGetAuthorizations) (*mtproto.Account_Authorizations, error) {
	rValue, err := c.svcCtx.Dao.AuthsessionClient.AuthsessionGetAuthorizations(c.ctx, &authsession.TLAuthsessionGetAuthorizations{
		UserId:           c.MD.UserId,
		ExcludeAuthKeyId: c.MD.AuthId,
	})
	if err != nil {
		c.Logger.Errorf("account.getAuthorizations - error: %v", err)
		return nil, err
	}

	return rValue, nil
}
