package core

import (
	"github.com/teamgram/proto/mtproto"
	"github.com/teamgram/teamgram-server/app/service/authsession/authsession"
	userpb "github.com/teamgram/teamgram-server/app/service/biz/user/user"
	"github.com/teamgram/teamgram-server/app/service/biz/user/user_password"
	usernamepb "github.com/teamgram/teamgram-server/app/service/biz/username/username"
	"golang.org/x/crypto/bcrypt"
)

// AuthUsernameSignIn
// auth.usernameSignIn username:string password:string = Auth_Authorization;
func (c *AuthorizationCore) AuthUsernameSignIn(in *mtproto.TLAuthUsernameSignIn) (*mtproto.Auth_Authorization, error) {
	if in.Username == "" || in.Password == "" {
		return nil, mtproto.ErrInputRequestInvalid
	}

	// 1. resolve username to user_id
	peer, err := c.svcCtx.Dao.UsernameClient.UsernameResolveUsername(c.ctx, &usernamepb.TLUsernameResolveUsername{
		Username: in.Username,
	})
	if err != nil || peer == nil {
		c.Logger.Errorf("username not found: %s", in.Username)
		return nil, mtproto.ErrUsernameNotOccupied
	}

	userId := peer.UserId
	if userId == 0 {
		c.Logger.Errorf("invalid user_id for username: %s", in.Username)
		return nil, mtproto.ErrUsernameNotOccupied
	}

	// 2. get password hash
	passwordResp, err := c.svcCtx.Dao.UserPasswordClient.GetUserPassword(c.ctx, &user_password.GetUserPasswordReq{
		UserId: userId,
	})
	if err != nil || passwordResp.PasswordHash == "" {
		c.Logger.Errorf("password not found for user: %d", userId)
		return nil, mtproto.ErrPasswordHashInvalid
	}

	// 3. verify password
	err = bcrypt.CompareHashAndPassword([]byte(passwordResp.PasswordHash), []byte(in.Password))
	if err != nil {
		c.Logger.Errorf("password mismatch for user: %d", userId)
		return nil, mtproto.ErrPasswordHashInvalid
	}

	// 4. get user info
	user, err := c.svcCtx.Dao.UserClient.UserGetImmutableUser(c.ctx, &userpb.TLUserGetImmutableUser{
		Id: userId,
	})
	if err != nil || user == nil {
		c.Logger.Errorf("get user info error: %v", err)
		return nil, mtproto.ErrInternelServerError
	}

	if user.Deleted() {
		c.Logger.Errorf("user %d is deleted, deny sign in", userId)
		return nil, mtproto.ErrUsernameNotOccupied
	}

	// 5. bind existing auth_key to user
	c.svcCtx.Dao.AuthsessionClient.AuthsessionBindAuthKeyUser(c.ctx, &authsession.TLAuthsessionBindAuthKeyUser{
		AuthKeyId: c.MD.AuthId,
		UserId:    userId,
	})

	c.Logger.Infof("user signed in via MTProto: id=%d, username=%s", userId, in.Username)

	return mtproto.MakeTLAuthAuthorization(&mtproto.Auth_Authorization{
		User: user.ToSelfUser(),
	}).To_Auth_Authorization(), nil
}
