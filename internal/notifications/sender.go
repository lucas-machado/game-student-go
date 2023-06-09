package notifications

import (
	"github.com/sendgrid/sendgrid-go"
	"github.com/sendgrid/sendgrid-go/helpers/mail"
	log "github.com/sirupsen/logrus"
)

type Sender struct {
	client *sendgrid.Client
}

func NewSender(client *sendgrid.Client) *Sender {
	return &Sender{
		client: client,
	}
}

func (s *Sender) SendRegistrationEmail(destinationEmail string) error {
	from := mail.NewEmail("Escola do Jogo", "no-reply@companyemail.com")
	subject := "Bem vindo a Escola do Jogo!"
	to := mail.NewEmail("Estudante", destinationEmail)
	plainTextContent := "Bem vindo a Escola do Jogo."
	htmlContent := "<strong>Obrigado!</strong>"
	message := mail.NewSingleEmail(from, subject, to, plainTextContent, htmlContent)
	response, err := s.client.Send(message)
	if err != nil {
		return err
	}

	if response.StatusCode != 202 {
		log.Errorf("failure sending registration email with sendgrid: %v", response.Body)
	}

	return nil
}
