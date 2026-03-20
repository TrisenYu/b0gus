package services
/// Last modified at 2026/03/20 星期五 17:26:07

import "net/mail"
// telnet-cli can connect to SMTP/POP3/Telenet server...

type Mail struct {
	From                 mail.Address        `json:"from"`              // The email address of the original sender.
	ReplyTo              []*mail.Address     `json:"replyTo,omitempty"` // The email address to which bounces (undeliverable notifications) are to be forwarded.
	To                   []mail.Address      `json:"to"`                // The email addresses of the recipients.
	Cc                   []*mail.Address     `json:"cc,omitempty"`      // The email addresses of the CC recipients.
	Bcc                  []*mail.Address     `json:"bcc,omitempty"`     // The email addresses of the BCC recipients.
	RawMime              []byte              `json:"rawMime,omitempty"` // The raw mime of the email.
	MessageId            string              `json:"messageId"`         // message id
	Subject              string              `json:"subject"`
	BodyText             string              `json:"bodyText,omitempty"`       // The text version of the email.
	BodyHTML             string              `json:"bodyHtml,omitempty"`       // The HTML version of the email.
	BodyInlinePart       []*MailBodyRaw      `json:"bodyInlinePart,omitempty"` // The raw inline content of the email.
	Headers              map[string][]string `json:"headers,omitempty"`        // The email headers. (one header can be specified multiple times with different values)
	Attachments          []*SmtpAttachment   `json:"attachments,omitempty"`
	SizeBytes            int64               `json:"sizeBytes"`            // The size of the email in bytes.
	SizeHtmlBodyBytes    int64               `json:"sizeHtmlBodyBytes"`    // The size of the HTML body in bytes.
	SizeInlineBytes      int64               `json:"sizeInlineBytes"`      // The size of the inline content in bytes.
	SizeAttachmentsBytes int64               `json:"sizeAttachmentsBytes"` // The size of the attachments in bytes.
	Timestamp            int64               `json:"timestamp"`            // since epoch in miliseconds
	Spam                 *string             `json:"spam,omitempty"`       // optional, spam verdict
	Virus                *string             `json:"virus,omitempty"`      // optional, virus verdict
	Spf                  *string             `json:"spf,omitempty"`        // optinal, spf verdict
	Dkim                 *string             `json:"dkim,omitempty"`       // optional, dkim verdict
	Dmarc                *string             `json:"dmarc"`                // optional, dmarc verdict
}

type MailBodyRaw struct {
	ContentID          string `json:"contentId,omitempty"`          // The content id of the raw email.
	ContentType        string `json:"contentType"`                  // The content type of the raw email.
	ContentDisposition string `json:"contentDisposition,omitempty"` // The content disposition of the raw email.
	Content            []byte `json:"content"`                      // The raw content of the email.
}

type SmtpAttachment struct {
	ContentType        string  `json:"contentType"`                  // The content type of the attachment.
	ContentDisposition string  `json:"contentDisposition,omitempty"` // The content disposition of the attachment.
	Filename           string  `json:"filename"`                     // The name of the attachment.
	Content            []byte  `json:"content"`                      // The content of the attachment.
	ContentURL         *string `json:"contentUrl,omitempty"`         // The content uri of the attachment.
	ContentID          string  `json:"contentId,omitempty"`          // The content id of the attachment.
}
