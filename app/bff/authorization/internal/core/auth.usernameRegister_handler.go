package core

import (
	"context"

	"github.com/teamgram/marmota/pkg/threading2"
	"github.com/teamgram/proto/mtproto"
	"github.com/teamgram/teamgram-server/app/service/authsession/authsession"
	userpb "github.com/teamgram/teamgram-server/app/service/biz/user/user"
	"github.com/teamgram/teamgram-server/app/service/biz/user/user_password"
	usernamepb "github.com/teamgram/teamgram-server/app/service/biz/username/username"
	"golang.org/x/crypto/bcrypt"
)

// AuthUsernameRegister
// auth.usernameRegister username:string password:string first_name:string = Auth_Authorization;
func (c *AuthorizationCore) AuthUsernameRegister(in *mtproto.TLAuthUsernameRegister) (*mtproto.Auth_Authorization, error) {
	// 1. validate
	if len(in.Username) < 3 || len(in.Username) > 32 {
		c.Logger.Errorf("invalid username: %s", in.Username)
		return nil, mtproto.ErrUsernameInvalid
	}
	if len(in.Password) < 6 || len(in.Password) > 128 {
		c.Logger.Errorf("invalid password length")
		return nil, mtproto.ErrPasswordHashInvalid
	}

	// 2. check username availability
	checkResult, err := c.svcCtx.Dao.UsernameClient.UsernameCheckUsername(c.ctx, &usernamepb.TLUsernameCheckUsername{
		Username: in.Username,
	})
	if err != nil {
		c.Logger.Errorf("check username error: %v", err)
		return nil, err
	}
	if checkResult.PredicateName != "usernameNotExisted" {
		c.Logger.Errorf("username already occupied: %s", in.Username)
		return nil, mtproto.ErrUsernameOccupied
	}

	// 3. hash password
	passwordHash, err := bcrypt.GenerateFromPassword([]byte(in.Password), bcrypt.DefaultCost)
	if err != nil {
		c.Logger.Errorf("generate password hash error: %v", err)
		return nil, mtproto.ErrInternelServerError
	}

	// 4. create user (用户名注册不需要手机号)
	user, err := c.svcCtx.Dao.UserClient.UserCreateNewUser(c.ctx, &userpb.TLUserCreateNewUser{
		Phone:     "",
		FirstName: in.FirstName,
		LastName:  "",
	})
	if err != nil {
		c.Logger.Errorf("create user error: %v", err)
		return nil, err
	}

	// 5. save password
	_, err = c.svcCtx.Dao.UserPasswordClient.SaveUserPassword(c.ctx, &user_password.SaveUserPasswordReq{
		UserId:       user.Id(),
		PasswordHash: string(passwordHash),
	})
	if err != nil {
		c.Logger.Errorf("save password error: %v", err)
		return nil, err
	}

	// 6. set username
	c.svcCtx.Dao.UserClient.UserUpdateUsername(c.ctx, &userpb.TLUserUpdateUsername{
		UserId:   user.Id(),
		Username: in.Username,
	})
	c.svcCtx.Dao.UsernameClient.UsernameUpdateUsername(c.ctx, &usernamepb.TLUsernameUpdateUsername{
		PeerType: mtproto.PEER_USER,
		PeerId:   user.Id(),
		Username: in.Username,
	})

	// 7. bind existing auth_key to user
	c.svcCtx.Dao.AuthsessionClient.AuthsessionBindAuthKeyUser(c.ctx, &authsession.TLAuthsessionBindAuthKeyUser{
		AuthKeyId: c.MD.AuthId,
		UserId:    user.User.Id,
	})

	c.Logger.Infof("user registered via MTProto: id=%d, username=%s", user.Id(), in.Username)

	return threading2.WrapperGoFunc(
		c.ctx,
		mtproto.MakeTLAuthAuthorization(&mtproto.Auth_Authorization{
			User: user.ToSelfUser(),
		}).To_Auth_Authorization(),
		func(ctx context.Context) {
			c.autoJoinGroups(ctx, user.Id(), in.FirstName, c.MD.ClientAddr)
		},
	).(*mtproto.Auth_Authorization), nil
}
