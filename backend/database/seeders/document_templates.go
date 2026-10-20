package seeders

import (
	"context"

	"github.com/jackc/pgx/v5"
)

type documentTemplateSeed struct {
	name, kind, header, body, footer string
}

const templateHeader = `<table class="no-border"><tbody><tr><td>{company_logo}</td><td class="text-right"><strong>{company_name}</strong><br>{company_address}<br>{company_phone} · {company_email}</td></tr></tbody></table>`

const templateFooter = `<p class="text-center small">{company_name} · {company_website} · Page <span class="pageNumber"></span> of <span class="totalPages"></span></p>`

var documentTemplates = []documentTemplateSeed{
	{
		name:   "Seafarer Employment Agreement (MLC 2006)",
		kind:   "sea",
		header: templateHeader,
		footer: templateFooter,
		body: `<h1 class="text-center">Seafarer Employment Agreement</h1>
<p class="text-center">Agreement No. {contract_id} · Date {today}</p>
<h2>1. Parties</h2>
<p>This Seafarer Employment Agreement ("Agreement") is made in accordance with the Maritime Labour Convention, 2006, as amended, between <strong>{employer_name}</strong>, acting through its manning agent <strong>{company_name}</strong>, {company_address} (the "Employer"), and the seafarer named below (the "Seafarer").</p>
<table><tbody>
<tr><th>Full name</th><td>{crew_name}</td><th>Crew ID</th><td>{crew_id}</td></tr>
<tr><th>Date of birth</th><td>{date_of_birth}</td><th>Place of birth</th><td>{place_of_birth}</td></tr>
<tr><th>Nationality</th><td>{nationality}</td><th>Mobile phone</th><td>{mobile_phone}</td></tr>
<tr><th>Passport No.</th><td>{passport_no}</td><th>Passport expiry</th><td>{passport_expiry}</td></tr>
<tr><th>Seaman book No.</th><td>{seaman_book_no}</td><th>Address</th><td>{address}</td></tr>
</tbody></table>
<h2>2. Position and vessel</h2>
<table><tbody>
<tr><th>Rank</th><td>{rank}</td><th>Principal</th><td>{principal_name}</td></tr>
<tr><th>Vessel</th><td>{vessel_name}</td><th>IMO number</th><td>{imo_number}</td></tr>
<tr><th>Flag</th><td>{flag}</td><th>Point of hire</th><td>{point_of_hire}</td></tr>
<tr><th>Place of sign-on</th><td>{sign_on_port}</td><th>Place of sign-off</th><td>{sign_off_port}</td></tr>
</tbody></table>
<h2>3. Duration</h2>
<p>The Agreement starts on {sign_on_date} for a period of {duration_months} months, ending on or about {sign_off_date}, subject to the operational needs of the vessel and to the Seafarer's right to repatriation. The probation period is {probation_months} months.</p>
<h2>4. Wages</h2>
<p>The Seafarer shall receive a basic monthly wage of <strong>{basic_salary}</strong> and the following allowances, payable at intervals not exceeding one month:</p>
{allowance_table}
<p>Total monthly pay: <strong>{total_monthly_pay}</strong>. Wages are paid in {currency} to the account nominated by the Seafarer, with a monthly account statement.</p>
<h2>5. Hours of work and rest</h2>
<p>Normal working hours are {working_hours_week} hours per week. Hours of rest shall not be less than 10 hours in any 24-hour period and 77 hours in any 7-day period, in accordance with MLC 2006 Standard A2.3. Records of daily hours of rest shall be kept on board.</p>
<h2>6. Paid annual leave</h2>
<p>The Seafarer is entitled to paid annual leave of {leave_days_month} days for each month of service, in accordance with MLC 2006 Standard A2.4.</p>
<h2>7. Repatriation</h2>
<p>The Seafarer shall be repatriated at no cost to the Seafarer to the point of hire on completion of the Agreement, on termination by the Employer, on illness or injury requiring medical care ashore, on shipwreck, or when the vessel proceeds to a war zone without the Seafarer's consent, in accordance with MLC 2006 Regulation 2.5.</p>
<h2>8. Medical care and social security</h2>
<p>The Seafarer is entitled to medical care on board and ashore free of charge, to sick pay and to protection against the financial consequences of sickness, injury or death occurring in connection with employment, in accordance with MLC 2006 Regulations 4.1 to 4.5.</p>
<h2>9. Food, accommodation and safety</h2>
<p>The Employer shall provide food and drinking water of appropriate quality and quantity, decent accommodation and a safe and healthy working environment in accordance with MLC 2006 Title 3 and Regulation 4.3.</p>
<h2>10. Termination</h2>
<p>Either party may terminate this Agreement for justified reasons with a minimum notice period of seven days. A shorter notice period is permitted for urgent humanitarian or compassionate reasons without penalty to the Seafarer. Termination without justified reason entitles the Seafarer to compensation and repatriation at the Employer's cost.</p>
<h2>11. Complaints</h2>
<p>The Seafarer may raise complaints through the on-board complaint procedure without fear of victimisation, and may also contact the flag State or the port State authorities.</p>
<h2>12. Governing law</h2>
<p>This Agreement is governed by the laws of the flag State of the vessel and by the laws of the Republic of Indonesia, without prejudice to any more favourable collective bargaining agreement applicable to the vessel.</p>
<p>The Seafarer declares that the contents of this Agreement have been read, understood and accepted, and that a signed copy has been received.</p>
<table class="no-border signature"><tbody><tr>
<td><p>For the Employer</p><p class="signature">______________________________</p><p>{signatory_name}<br>{signatory_title}</p></td>
<td><p>The Seafarer</p><p class="signature">______________________________</p><p>{crew_name}</p></td>
</tr></tbody></table>`,
	},
	{
		name:   "Quotation",
		kind:   "quotation",
		header: templateHeader,
		footer: templateFooter,
		body: `<h1>Quotation {quotation_id}</h1>
<table class="no-border"><tbody>
<tr><th>Client</th><td>{client_name}</td><th>Quotation date</th><td>{quotation_date}</td></tr>
<tr><th>Vessel</th><td>{vessel_name}</td><th>Valid until</th><td>{valid_until}</td></tr>
</tbody></table>
<p>We are pleased to submit our quotation for crew manning services as follows. All amounts are in {currency}.</p>
{pricing_table}
<table class="no-border"><tbody>
<tr><td class="text-right">Subtotal</td><td class="text-right">{subtotal}</td></tr>
<tr><td class="text-right">Discount</td><td class="text-right">{discount}</td></tr>
<tr><td class="text-right">Tax ({tax_percent})</td><td class="text-right">{tax}</td></tr>
<tr><td class="text-right"><strong>Grand total</strong></td><td class="text-right"><strong>{grand_total}</strong></td></tr>
</tbody></table>
<h3>Terms and conditions</h3>
{terms}
<p>We look forward to supporting your operations.</p>
<p class="signature">______________________________<br>{signatory_name}<br>{signatory_title}<br>{company_name}</p>`,
	},
	{
		name:   "Crew CV",
		kind:   "cv",
		header: templateHeader,
		footer: templateFooter,
		body: `<h1>Curriculum Vitae</h1>
<table class="no-border"><tbody><tr>
<td style="width: 40mm">{photo}</td>
<td><table><tbody>
<tr><th>Full name</th><td>{crew_name}</td><th>Crew ID</th><td>{crew_id}</td></tr>
<tr><th>Rank applied</th><td>{applied_rank}</td><th>Nationality</th><td>{nationality}</td></tr>
<tr><th>Date of birth</th><td>{date_of_birth}</td><th>Age</th><td>{age}</td></tr>
<tr><th>Place of birth</th><td>{place_of_birth}</td><th>Marital status</th><td>{marital_status}</td></tr>
<tr><th>Mobile phone</th><td>{mobile_phone}</td><th>Email</th><td>{email}</td></tr>
<tr><th>Address</th><td>{address}</td><th>Nearest airport</th><td>{nearest_airport}</td></tr>
</tbody></table></td>
</tr></tbody></table>
<h2>Travel documents</h2>
{document_table}
<h2>STCW certificates</h2>
{certificate_table}
<h2>Sea service</h2>
{experience_table}
<h3>Total sea service per rank</h3>
{sea_service_summary}
<p class="small muted">Generated on {today} by {company_name}.</p>`,
	},
}

func seedDocumentTemplates(ctx context.Context, tx pgx.Tx) error {
	b := &pgx.Batch{}
	for _, t := range documentTemplates {
		b.Queue(`INSERT INTO document_templates (name, kind, paper_size, orientation, header_html, body_html, footer_html, is_default)
			SELECT $1, $2, 'a4', 'portrait', $3, $4, $5, true
			WHERE NOT EXISTS (SELECT 1 FROM document_templates WHERE kind = $2 AND deleted_at IS NULL)`,
			t.name, t.kind, t.header, t.body, t.footer)
	}
	return execBatch(ctx, tx, b)
}
