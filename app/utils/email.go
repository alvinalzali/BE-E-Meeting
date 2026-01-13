package utils

import (
	"crypto/tls"
	"fmt"
	"os"
	"strconv"

	"gopkg.in/gomail.v2"
)

func SendResetEmail(toEmail, resetToken string) error {
	smtpHost := os.Getenv("SMTP_HOST")
	smtpPortStr := os.Getenv("SMTP_PORT")
	smtpUser := os.Getenv("SMTP_USER")
	smtpPass := os.Getenv("SMTP_PASS")
	smtpFrom := os.Getenv("SMTP_FROM")
	baseURL := os.Getenv("APP_URL")

	// [DEBUG] Cek ENV Email
	fmt.Println("--- DEBUG EMAIL ---")
	fmt.Printf("Host: %s, Port: %s, User: %s\n", smtpHost, smtpPortStr, smtpUser)

	smtpPort, err := strconv.Atoi(smtpPortStr)
	if err != nil {
		fmt.Println("Error: Port bukan angka")
		return err
	}

	// link reset
	resetLink := fmt.Sprintf("%s/password/reset/%s", baseURL, resetToken)

	m := gomail.NewMessage()
	m.SetHeader("From", smtpFrom)
	m.SetHeader("To", toEmail)
	m.SetHeader("Subject", "Reset Password Request")

	htmlBody := fmt.Sprintf(`
    <h1>Reset Password Request</h1>
    <p>Click the link below to reset your password:</p>
    <p><a href="%s">%s</a></p>
    <br>
    <p>This link will expire in 15 minutes.</p>
    `, resetLink, resetLink)

	m.SetBody("text/html", htmlBody)

	d := gomail.NewDialer(smtpHost, smtpPort, smtpUser, smtpPass)
	d.TLSConfig = &tls.Config{InsecureSkipVerify: true}

	fmt.Println("Sending email link reset to", toEmail)
	fmt.Println("Reset token :", resetToken)

	return d.DialAndSend(m)
}
