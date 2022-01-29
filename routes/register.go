package routes

import (
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
	email2 "github.com/uberswe/golang-base-project/email"
	"github.com/uberswe/golang-base-project/models"
	"github.com/uberswe/golang-base-project/ulid"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
	"log"
	"net/http"
	"net/url"
	"path"
	"time"
)

// Register renders the HTML content of the register page
func (controller Controller) Register(c *gin.Context) {
	pd := controller.DefaultPageData(c)
	pd.Title = pd.Trans("Register")
	c.HTML(http.StatusOK, "register.html", pd)
}

// RegisterPost handles requests to register users and returns appropriate messages as HTML content
func (controller Controller) RegisterPost(c *gin.Context) {
	pd := controller.DefaultPageData(c)
	passwordError := pd.Trans("Your password must be 8 characters in length or longer")
	registerError := pd.Trans("Could not register, please make sure the details you have provided are correct and that you do not already have an existing account.")
	registerSuccess := pd.Trans("Thank you for registering. An activation email has been sent with steps describing how to activate your account.")
	pd.Title = pd.Trans("Register")
	password := c.PostForm("password")
	if len(password) < 8 {
		pd.Messages = append(pd.Messages, Message{
			Type:    "error",
			Content: passwordError,
		})
		c.HTML(http.StatusBadRequest, "register.html", pd)
		return
	}

	// The password is hashed as early as possible to make timing attacks that reveal registered users harder
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		pd.Messages = append(pd.Messages, Message{
			Type:    "error",
			Content: registerError,
		})
		log.Println(err)
		c.HTML(http.StatusInternalServerError, "register.html", pd)
		return
	}

	email := c.PostForm("email")

	// Validate the email
	validate := validator.New()
	err = validate.Var(email, "required,email")

	if err != nil {
		pd.Messages = append(pd.Messages, Message{
			Type:    "error",
			Content: registerError,
		})
		log.Println(err)
		c.HTML(http.StatusInternalServerError, "register.html", pd)
		return
	}

	user := models.User{Email: email}

	res := controller.db.Where(&user).First(&user)
	if (res.Error != nil && res.Error != gorm.ErrRecordNotFound) || res.RowsAffected > 0 {
		pd.Messages = append(pd.Messages, Message{
			Type:    "error",
			Content: registerError,
		})
		log.Println(res.Error)
		c.HTML(http.StatusInternalServerError, "register.html", pd)
		return
	}

	if err != nil {
		pd.Messages = append(pd.Messages, Message{
			Type:    "error",
			Content: registerError,
		})
		log.Println(err)
		c.HTML(http.StatusInternalServerError, "register.html", pd)
		return
	}

	user.Password = string(hashedPassword)

	res = controller.db.Save(&user)
	if res.Error != nil || res.RowsAffected == 0 {
		pd.Messages = append(pd.Messages, Message{
			Type:    "error",
			Content: registerError,
		})
		log.Println(res.Error)
		c.HTML(http.StatusInternalServerError, "register.html", pd)
		return
	}

	// Generate activation token and send activation email
	go controller.activationEmailHandler(user.ID, email, pd.Trans)

	pd.Messages = append(pd.Messages, Message{
		Type:    "success",
		Content: registerSuccess,
	})

	c.HTML(http.StatusOK, "register.html", pd)
}

func (controller Controller) activationEmailHandler(userID uint, email string, trans func(string) string) {
	activationToken := models.Token{
		Value: ulid.Generate(),
		Type:  models.TokenUserActivation,
	}

	res := controller.db.Where(&activationToken).First(&activationToken)
	if (res.Error != nil && res.Error != gorm.ErrRecordNotFound) || res.RowsAffected > 0 {
		// If the activation token already exists we try to generate it again
		controller.activationEmailHandler(userID, email, trans)
		return
	}

	activationToken.ModelID = int(userID)
	activationToken.ModelType = "User"
	activationToken.ExpiresAt = time.Now().Add(time.Minute * 10)

	res = controller.db.Save(&activationToken)
	if res.Error != nil || res.RowsAffected == 0 {
		log.Println(res.Error)
		return
	}
	controller.sendActivationEmail(activationToken.Value, email, trans)
}

func (controller Controller) sendActivationEmail(token string, email string, trans func(string) string) {
	u, err := url.Parse(controller.config.BaseURL)
	if err != nil {
		log.Println(err)
		return
	}

	u.Path = path.Join(u.Path, "/activate/", token)

	activationURL := u.String()

	emailService := email2.New(controller.config)

	emailService.Send(email, trans("User Activation"), fmt.Sprintf(trans("Use the following link to activate your account. If this was not requested by you, please ignore this email.\n%s"), activationURL))
}
