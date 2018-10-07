package notification

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"syscall"
	"text/template"
)

const (
	BalloonTipIconError   = "Error"
	BalloonTipIconWarning = "Warning"
	BalloonTipIconInfo    = "Info"
)

const script = `
[void] [System.Reflection.Assembly]::LoadWithPartialName("System.Windows.Forms")
$n = New-Object System.Windows.Forms.NotifyIcon
$n.Icon = "icon.ico"
$n.BalloonTipIcon = "{{.BalloonTipIcon}}"
$n.BalloonTipText = "{{.BalloonTipText}}"
$n.BalloonTipTitle = "{{.BalloonTipTitle}}"
$n.Text = "{{.BalloonTipText}}"
$n.Visible = $True
$n.ShowBalloonTip(0)
`

// Notification is a Windows notification.
type notification struct {
	BalloonTipIcon  string
	BalloonTipText  string
	BalloonTipTitle string
}

func Info(title string, text string) error {
	return send(notification{BalloonTipIconInfo, text, title})
}

func Warning(title string, text string) error {
	return send(notification{BalloonTipIconWarning, text, title})
}

func Error(title string, text string) error {
	return send(notification{BalloonTipIconError, text, title})
}

func send(n notification) error {
	if len(n.BalloonTipText) > 61 {
		n.BalloonTipText = n.BalloonTipText[:61] + "..."
	}

	tmpl, err := template.New("").Parse(script)
	if err != nil {
		return err
	}

	buf := new(bytes.Buffer)
	if err := tmpl.Execute(buf, n); err != nil {
		return err
	}

	fmt.Printf("%s", buf)

	cmd := exec.Command("PowerShell", "-Command", buf.String())
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	return cmd.Run()
}
