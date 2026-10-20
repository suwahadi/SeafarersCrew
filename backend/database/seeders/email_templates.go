package seeders

type emailTemplateSeed struct {
	code, name, subject, body string
}

const emailSignature = `<p>Kind regards,<br>{company_name}<br>{company_phone} · {company_email}</p>`

var emailTemplates = []emailTemplateSeed{
	{
		code:    "send_contract",
		name:    "Send Contract",
		subject: "Seafarer Employment Agreement {contract_id} – {crew_name}",
		body: `<p>Dear {crew_name},</p>
<p>Please find attached your Seafarer Employment Agreement <strong>{contract_id}</strong> for the position of <strong>{rank}</strong> on board <strong>{vessel_name}</strong>, with sign-on on {sign_on_date}.</p>
<p>Kindly review the agreement and contact us if you have any questions before signing.</p>` + emailSignature,
	},
	{
		code:    "send_quotation",
		name:    "Send Quotation",
		subject: "Quotation {quotation_id} from {company_name}",
		body: `<p>Dear {client_name},</p>
<p>Thank you for your interest in our crew manning services. Please find attached our quotation <strong>{quotation_id}</strong> dated {quotation_date}, valid until {valid_until}.</p>
<p>We look forward to supporting your operations.</p>` + emailSignature,
	},
	{
		code:    "send_cv",
		name:    "Send CV",
		subject: "Crew CV – {crew_name} ({rank})",
		body: `<p>Dear Sir or Madam,</p>
<p>Please find attached the curriculum vitae of <strong>{crew_name}</strong> (Crew ID {crew_id}) for your consideration.</p>
<p>We remain at your disposal for any further information.</p>` + emailSignature,
	},
	{
		code:    "applicant_confirmation",
		name:    "Applicant Confirmation",
		subject: "We received your application {application_id}",
		body: `<p>Dear {applicant_name},</p>
<p>Thank you for applying for <strong>{vacancy_title}</strong>. Your application ID is <strong>{application_id}</strong>.</p>
<p>Our recruitment team will review your application and contact you if your profile matches the requirements. {company_name} never charges applicants any fee.</p>` + emailSignature,
	},
	{
		code:    "admin_new_application",
		name:    "Admin – New Application",
		subject: "New application {application_id} for {vacancy_title}",
		body: `<p>A new application was submitted.</p>
<ul><li>Application ID: {application_id}</li><li>Vacancy: {vacancy_title} ({vacancy_id})</li><li>Applicant: {applicant_name}</li><li>Email: {applicant_email}</li></ul>
<p><a href="{admin_url}">Open the applicant record</a></p>`,
	},
	{
		code:    "admin_new_contact",
		name:    "Admin – New Contact Message",
		subject: "New contact message: {subject}",
		body: `<p>A new message was sent through the contact form.</p>
<ul><li>Name: {sender_name}</li><li>Email: {sender_email}</li><li>Phone: {sender_phone}</li></ul>
<p>{message}</p>
<p><a href="{admin_url}">Open the contact inbox</a></p>`,
	},
}
