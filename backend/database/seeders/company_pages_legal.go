package seeders

const termsHTML = `<h2>1. Use of this website</h2>
<p>These Terms and Conditions govern your use of the website of {{legal_name}} ("{{name}}", "we", "us"). By using this website you agree to these terms. If you do not agree, please do not use the website.</p>
<ul>
<li>The website is provided to inform visitors about our crew manning services and to receive career applications and enquiries.</li>
<li>You agree to use the website lawfully and not to disrupt it, attempt to gain unauthorised access, submit malicious content or use automated tools to collect data from it.</li>
<li>We may change, suspend or withdraw any part of the website at any time without notice.</li>
</ul>
<h2>2. Career applications</h2>
<p>When you apply for a vacancy through this website you confirm that:</p>
<ul>
<li>all information and documents you provide are accurate, complete and genuine, and you are entitled to submit them;</li>
<li>you understand that false, altered or misleading information or documents may lead to the rejection of your application or the end of any later engagement;</li>
<li>you have read our Privacy Policy and consent to us processing your application data for recruitment as described there.</li>
</ul>
<p>Submitting an application does not guarantee an interview, an offer or employment. Employment is only established by a written contract signed by you and the employer or principal concerned.</p>
<h3>No fees charged to applicants</h3>
<p>{{name}} does not charge applicants any fee at any stage of recruitment. Applying, screening, interviewing, document handling and placement are free of charge for seafarers. If anyone asks you for money in connection with a vacancy in our name, please do not pay and report it to us at <a href="mailto:{{email}}">{{email}}</a>.</p>
<h2>3. Intellectual property</h2>
<p>The content of this website, including text, images, logos, graphics and layout, belongs to {{legal_name}} or its licensors and is protected by applicable intellectual property laws. You may view and print pages for your personal, non-commercial use. You may not copy, reproduce, modify, distribute or republish any content without our prior written permission.</p>
<h2>4. Links to other websites</h2>
<p>This website may contain links to websites operated by third parties, such as clients, industry bodies or authorities. We do not control those websites and are not responsible for their content, availability or privacy practices. A link does not mean that we endorse the linked website.</p>
<h2>5. Limitation of liability</h2>
<p>We aim to keep the information on this website accurate and up to date, but we provide it on an "as is" and "as available" basis without warranties of any kind. To the fullest extent permitted by law, {{legal_name}} is not liable for any loss or damage arising from your use of, or inability to use, this website or from reliance on its content, including interruptions, errors, omissions or the content of third-party websites. Nothing in these terms excludes liability that cannot be excluded under applicable law.</p>
<h2>6. Governing law</h2>
<p>These Terms and Conditions are governed by and interpreted in accordance with the laws of the Republic of Indonesia. Any dispute arising from them or from your use of this website shall first be resolved through discussion in good faith and, failing that, shall be submitted to the courts of competent jurisdiction in the Republic of Indonesia.</p>
<h2>7. Changes to these terms</h2>
<p>We may update these Terms and Conditions from time to time. The current version is always published on this page, and your continued use of the website after a change means that you accept the updated terms.</p>
<h2>8. Contact</h2>
<p>If you have questions about these terms, please contact {{legal_name}} at <a href="mailto:{{email}}">{{email}}</a> or write to us at {{address}}.</p>`

const privacyHTML = `<h2>1. Who is responsible for your data</h2>
<p>{{legal_name}} ("{{name}}", "we", "us"), {{address}}, is the data controller for personal data collected through this website. You can reach us at <a href="mailto:{{email}}">{{email}}</a>.</p>
<h2>2. Data we collect</h2>
<h3>Contact form</h3>
<p>Your name, email address, phone number (optional), subject and message, together with the IP address and the date and time of submission.</p>
<h3>Career application form</h3>
<ul>
<li>Your full name, email address, phone number, date of birth, nationality and the rank you apply for;</li>
<li>your years of experience and an optional cover message;</li>
<li>the files you upload, such as your CV and supporting documents (for example seafarer certificates or identity and travel documents you choose to attach);</li>
<li>the IP address and time of your submission and the time you gave your consent.</li>
</ul>
<h3>Technical data</h3>
<p>Our servers process standard technical data such as IP address and request logs to keep the website secure and available. Our logs do not record the content of your applications.</p>
<h2>3. Why we use your data</h2>
<ul>
<li><strong>Recruitment:</strong> to assess your application, contact you, arrange interviews and, where you are selected, prepare employment.</li>
<li><strong>Communication:</strong> to answer your enquiries and send you messages about your application or request.</li>
<li><strong>Security and quality:</strong> to prevent spam and abuse of our forms and to keep the website running reliably.</li>
</ul>
<p>We do not use your data for advertising and we do not sell it.</p>
<h2>4. Legal basis and consent</h2>
<p>We process your application data on the basis of your consent, which you give by ticking the consent box when you submit the form, and as necessary to take steps at your request before entering into an employment relationship. We process contact form data to answer your enquiry and, for security data, on the basis of our legitimate interest in protecting our website. You may withdraw your consent at any time as described in section 8; withdrawal does not affect processing carried out before it.</p>
<h2>5. How long we keep your data</h2>
<ul>
<li><strong>Applicant data</strong> (application details and uploaded files) is kept for 24 months after your application, after which it is deleted or irreversibly anonymised, unless you ask us to delete it earlier or a longer period is required by law.</li>
<li><strong>Contact messages</strong> are kept for as long as needed to handle your enquiry and for our reasonable record-keeping afterwards.</li>
<li><strong>Technical logs</strong> are kept only for a short period needed for security and troubleshooting.</li>
</ul>
<h2>6. Who we share your data with</h2>
<p>We share application data with the shipping companies and principals we recruit for only when it is needed to consider you for a vacancy or placement. We do not share it with them for any other purpose.</p>
<p>We also use service providers that process data on our behalf under our instructions, such as hosting and file storage, email delivery and bot protection. They may only use your data to provide their services to us. We may disclose data where required by law or by a competent authority.</p>
<h2>7. Storage and security</h2>
<p>Uploaded files are stored privately and are only opened through short-lived, access-controlled links. Access to applicant data is limited to authorised staff and protected by individual accounts and secure sessions, and data is transmitted over encrypted connections. No system is completely secure, but we take reasonable technical and organisational measures to protect your data against loss, misuse and unauthorised access.</p>
<h2>8. Your rights under Indonesian law</h2>
<p>Under Law No. 27 of 2022 on Personal Data Protection (UU PDP) you have the right to:</p>
<ul>
<li>obtain information about how your personal data is processed;</li>
<li>access your personal data and obtain a copy of it;</li>
<li>complete, update or correct your personal data;</li>
<li>end the processing of, delete or destroy your personal data;</li>
<li>withdraw the consent you have given;</li>
<li>object to certain automated decisions, and request that processing be suspended or restricted in the cases the law allows;</li>
<li>file a complaint and claim redress for a breach of your personal data protection rights.</li>
</ul>
<p>To exercise these rights, contact us as described in section 10. We may need to verify your identity before acting on a request.</p>
<h2>9. Cookies</h2>
<p>This website uses only the cookies and similar technologies it needs to work:</p>
<ul>
<li><strong>Language preference:</strong> remembers the language you choose in the language switcher.</li>
<li><strong>Cloudflare Turnstile:</strong> a bot-protection check on our contact and application forms, which may set a technical cookie or read browser signals to tell people from automated programs.</li>
</ul>
<p>We do not use advertising or cross-site tracking cookies. You can block cookies in your browser settings, but the language switcher and the forms may then not work properly.</p>
<h2>10. Contact for privacy requests</h2>
<p>For any question about this policy or to exercise your rights, email <a href="mailto:{{email}}">{{email}}</a> or write to {{legal_name}}, {{address}}. We will respond as soon as reasonably possible and within the time limits set by law.</p>
<h2>11. Changes to this policy</h2>
<p>We may update this Privacy Policy when our practices or the law change. The current version is always published on this page.</p>`

type faqSeed struct {
	question, answer string
}

var faqItems = []faqSeed{
	{
		"How do I apply for a job?",
		`<p>Open the <a href="/career">Career page</a>, choose a vacancy that matches your rank and experience and complete the online application form. Attach your CV and, if you can, your supporting documents, then submit. You can apply for more than one vacancy; if you apply to the same vacancy again with the same email address within 30 days we treat it as a duplicate.</p><p>After you apply, your application moves through these stages: New, Screening, Interview, and finally Accepted or Rejected.</p>`,
	},
	{
		"Which documents do I need to apply?",
		`<p>At minimum we need an up-to-date CV. For most positions we will also ask for your passport, seaman's book (CDC), medical check-up (MCU) result and the STCW certificates required for your rank, such as Basic Safety Training, together with your certificate of competency if you are an officer. The required documents are listed on each vacancy. You can send additional documents later when our crewing team asks for them.</p>`,
	},
	{
		"Do I have to pay any fee?",
		`<p>No. {{name}} never charges applicants any fee for applying, screening, interviews, document handling or placement. If someone asks you to pay money in our name, please do not pay and report it to <a href="mailto:{{email}}">{{email}}</a>.</p>`,
	},
	{
		"How long is a typical contract?",
		`<p>Contract length depends on the vessel, the rank and the principal. The expected duration is stated on each vacancy, and the exact period, together with your salary and allowances, is written in your seafarer employment agreement before you sign on. Offshore and shipping contracts are usually counted in months.</p>`,
	},
	{
		"Which positions do you recruit for?",
		`<p>We recruit for deck, engine and catering departments, from masters and chief engineers to ratings such as AB, OS, oiler, wiper, cook and steward. Open positions are published on the <a href="/career">Career page</a>.</p>`,
	},
	{
		"How will I hear about my application?",
		`<p>Our crewing team reviews each application and contacts you by email or phone if your profile matches the vacancy. Please make sure your email address and phone number are correct and check your spam folder. Due to the number of applications we may not be able to reply individually to every candidate.</p>`,
	},
	{
		"Do I need valid certificates before I apply?",
		`<p>Valid mandatory certificates are required before you can be deployed, and we check them against the requirements of the rank. You can apply while some of your documents are being renewed, but please tell us in your message which ones, and send the new copies as soon as you receive them.</p>`,
	},
	{
		"How long do you keep my personal data?",
		`<p>Applicant data is kept for 24 months after your application and is then deleted or anonymised. You can ask us to delete it earlier. See our <a href="/privacy-policy">Privacy Policy</a> for details.</p>`,
	},
	{
		"Who can see my application?",
		`<p>Only authorised members of our crewing team. We share your details with a shipping company or principal only when we consider you for one of their vacancies. Uploaded files are stored privately and opened only through short-lived secure links.</p>`,
	},
	{
		"How can I update or delete my information?",
		`<p>Email <a href="mailto:{{email}}">{{email}}</a> with your name and the vacancy you applied for, and tell us what you would like to correct or delete. We may ask you to confirm your identity first.</p>`,
	},
}
