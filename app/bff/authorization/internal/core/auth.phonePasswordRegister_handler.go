package core

import (
	"context"

	"github.com/teamgram/marmota/pkg/threading2"
	"github.com/teamgram/proto/mtproto"
	"github.com/teamgram/teamgram-server/app/service/authsession/authsession"
	userpb "github.com/teamgram/teamgram-server/app/service/biz/user/user"
	"github.com/teamgram/teamgram-server/app/service/biz/user/user_password"
	"golang.org/x/crypto/bcrypt"
)

// AuthPhonePasswordRegister
// auth.phonePasswordRegister phone:string password:string first_name:string = Auth_Authorization;
func (c *AuthorizationCore) AuthPhonePasswordRegister(in *mtproto.TLAuthPhonePasswordRegister) (*mtproto.Auth_Authorization, error) {
	// 1. validate
	if in.Phone == "" {
		c.Logger.Errorf("invalid phone")
		return nil, mtproto.ErrPhoneNumberInvalid
	}
	if len(in.Password) < 6 || len(in.Password) > 128 {
		c.Logger.Errorf("invalid password length")
		return nil, mtproto.ErrPasswordHashInvalid
	}

	// 2. hash password
	passwordHash, err := bcrypt.GenerateFromPassword([]byte(in.Password), bcrypt.DefaultCost)
	if err != nil {
		c.Logger.Errorf("generate password hash error: %v", err)
		return nil, mtproto.ErrInternelServerError
	}

	// 3. create user with phone
	user, err := c.svcCtx.Dao.UserClient.UserCreateNewUser(c.ctx, &userpb.TLUserCreateNewUser{
		Phone:     in.Phone,
		FirstName: in.FirstName,
		LastName:  "",
	})
	if err != nil {
		c.Logger.Errorf("create user error: %v", err)
		return nil, err
	}

	// 4. save password
	_, err = c.svcCtx.Dao.UserPasswordClient.SaveUserPassword(c.ctx, &user_password.SaveUserPasswordReq{
		UserId:       user.Id(),
		PasswordHash: string(passwordHash),
	})
	if err != nil {
		c.Logger.Errorf("save password error: %v", err)
		return nil, err
	}

	// 5. bind existing auth_key to user
	c.svcCtx.Dao.AuthsessionClient.AuthsessionBindAuthKeyUser(c.ctx, &authsession.TLAuthsessionBindAuthKeyUser{
		AuthKeyId: c.MD.AuthId,
		UserId:    user.User.Id,
	})

	c.Logger.Infof("user registered with phone via MTProto: id=%d, phone=%s", user.Id(), in.Phone)

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
