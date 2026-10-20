# Seafarers Crew — Product Requirements Document

Oct 21, 2026 · @Ambon

## 1. Document Control

This PRD (v2.0) is the build baseline for Seafarers Crew and supersedes the v1.0 Indonesian draft.

| Field | Value |
| --- | --- |
| Product name | Seafarers Crew |
| Document type | Product Requirements Document |
| Version | 2.0 |
| Status | Approved for build |
| Language | English |
| Audience | Product owner, backend, frontend and DevOps engineers, AI coding agents |
| Companion file | AGENTS.md (repository root) |

| Version | Change |
| --- | --- |
| 2.0 | Full rewrite in English. Adds Website CMS, Public Website, Career Portal, Contracts, Rotation Planning, Compliance Matrix, STCW Certificates, Brevo email, file pipeline, Admin Settings, engineering guidelines. |
| 1.0 | Initial Indonesian draft: master data, crew multi-tab form, contract builder, datagrid. |

## 2. Executive Summary & Product Goals

Seafarers Crew is a single-admin web platform that runs an offshore manning agency end to end: crew records, STCW compliance, contracts, rotation, quotations, and a public company website with a career portal.

It replaces spreadsheets and paper files with one secure system. The admin manages seafarer profiles, sea service history, travel documents and certificates, and generates contracts, quotations and crew CVs as branded PDFs. The public website promotes the company, publishes news, and collects job applications.

The product has three parts:

- **Part A, Crew Operations (Admin):** crew data, compliance, contracts, rotation, document builder, email dispatch, reporting.
- **Part B, Website CMS (Admin):** pages, news, sliders, gallery, testimonials, vacancies, applicants, contact inbox, languages.
- **Part C, Public Website (Next.js):** company profile, news, career portal, contact form, multi-language switcher.

### Product goals

| Goal | Success measure |
| --- | --- |
| Centralize crew data | 100% of active crew records, documents and certificates stored in the system |
| Prevent compliance gaps | Every document and certificate expiring within 90 days is visible on the dashboard |
| Speed up paperwork | A contract, quotation or CV is generated as PDF in under 10 seconds |
| Fast admin UX | Datagrid pages load in under 1.5 s with 50,000+ crew rows |
| Fast public site | Google PageSpeed Insights score of 85 or higher on mobile and desktop for every public page |
| Secure by default | No page, API or file is reachable without an authenticated session, except the public website and its forms |

## 3. Scope

Version 1 ships Parts A, B and C with one administrator account; audit log, 2FA and multi-role access move to the roadmap.

### In scope (v1)

- Single administrator authentication with Redis-backed sessions, CLI password reset and one-time recovery codes.
- Dashboard with KPIs and charts.
- Master data, including Master Vessel, Allowance Type, Document Type, Certificate Type and Compliance Matrix.
- Crew management with nine tabs, including STCW Certificates and Compliance Status.
- Contract management with multi-currency salary and allowances, and automatic sign-off into Experiences.
- Crew rotation planning (relief schedule).
- Document builder for Seafarer Employment Agreement, Quotation (daily rate and lump sum) and Crew CV, with PDF export.
- Email dispatch of generated PDFs through the Brevo API.
- Advanced datagrid, CSV/Excel export and CSV import.
- Website CMS: page builder, news, hero slider, gallery, testimonials, client logos, stats counter, job vacancies, applicant pipeline, contact inbox, language list.
- Public website on Next.js with SSR, career portal, contact form, Cloudflare Turnstile, Google Translate switcher, full SEO.
- File pipeline: validation, WebP and PDF compression, DOC/DOCX to PDF conversion, random renaming, private S3 storage.
- Admin Settings Panel.
- Daily automated PostgreSQL backup to S3.
- Docker Compose deployment and seed data.

### Out of scope (v1)

- Multiple users, roles and permissions.
- Audit log of data changes.
- Two-factor authentication.
- Email or WhatsApp alerts for expiring documents (alerts are visual, in-app only).
- Currency conversion and exchange rates.
- Antivirus scanning of uploads.
- Payroll processing, deductions and allotments.
- Crew self-service portal and applicant login.
- Converting applicants into crew records.
- Native mobile apps.

### Future roadmap

| Item | Reason deferred |
| --- | --- |
| Audit log | Needed once more than one user exists |
| Two-factor authentication | Extra hardening after launch |
| Multi-user and roles | Not required while one admin operates the system |
| Email digest of expiring documents | Brevo is already integrated, so this is a small addition |
| Applicant to crew conversion | Applicants stay a separate archive in v1 |
| Paid translation API | Replaces the unofficial Google widget if it breaks |

## 4. System Architecture & Tech Stack

Seafarers Crew is one Go API behind a Caddy proxy, serving a React admin SPA and a Next.js public site, with PostgreSQL, Redis, S3 and Gotenberg as internal services.

&#91;embedded content: System architecture · 8 Docker services, 2 external APIs\]

Both frontends talk only to the Go API; no frontend reaches the database, Redis or storage directly.

| Layer | Technology |
| --- | --- |
| Backend | Go 1.27, chi router, pgx + sqlc, golang-migrate, slog, govips (libvips), Ghostscript |
| Admin frontend | React 19, Vite, TypeScript, TanStack Query and Table, React Hook Form + Zod, TipTap, Tailwind CSS, Recharts |
| Public frontend | Next.js (App Router, SSR), TypeScript, Tailwind CSS |
| Database | PostgreSQL 17 |
| Cache, sessions, queue | Redis 7 |
| File storage | AWS S3 or any S3-compatible store (MinIO bundled) |
| PDF and conversion | Gotenberg 8 (Chromium for HTML to PDF, LibreOffice for DOC/DOCX to PDF) |
| Email | Brevo transactional API |
| Bot protection | Cloudflare Turnstile |
| Translation | Google Translate website widget (public site only) |
| Reverse proxy | Caddy 2 |
| Deployment | Docker Compose |

## 5. Authentication & Session Security

One administrator signs in with email and password; every admin page, admin API call and stored file requires a valid server-side session.

### Login

- Login with email and password. Passwords are hashed with Argon2id.
- Login form is protected by rate limiting: 5 failed attempts per IP per 15 minutes, then a 15-minute lockout.
- Generic error message on failure (no hint whether the email or the password was wrong).

### Session

- On login, the API creates a session ID of 32 random bytes (crypto/rand), stored in Redis under a unique key `sess:<sha256(session_id)>` with admin ID, IP, user agent and timestamps.
- The browser receives the session ID only in a cookie with `HttpOnly`, `Secure` and `SameSite=Strict`.
- Idle timeout: 8 hours. Each request refreshes the Redis TTL.
- Absolute lifetime: 7 days, after which the admin must log in again.
- The session ID is rotated on login and after a password change.
- Logout deletes the Redis key. A password change deletes all sessions of the admin.
- All state-changing requests require a CSRF token (double-submit pattern) in addition to the cookie.

### Access control

- Every route under `/api/admin/*` passes through the auth middleware; no admin data is served without a valid session.
- The React admin SPA checks `/api/admin/me` on load and redirects to `/login` when it returns 401.
- Stored files are never public. The API issues S3 presigned URLs valid for 5 minutes, only to authenticated sessions.
- Public API routes (`/api/public/*`) expose only published CMS content and accept only form submissions.

### Password recovery

Email is not used for password recovery. Two channels exist:

1. **CLI command:** `docker compose exec api ./seafarers admin:reset-password --email=<email>` prints a new temporary password and forces a change on next login.
2. **Recovery codes:** 10 one-time codes are shown once at initial setup. Each code is stored as an Argon2id hash and is invalidated after use. The admin can regenerate the set from Settings, which invalidates the old set.

### Initial setup

On first boot with no admin account, the CLI command `admin:create` creates the admin and prints the recovery codes. The web UI exposes no sign-up page.

## 6. Part A: Crew Operations (Admin)

Part A covers the daily work of the manning agency: knowing who is available, who is compliant, who is on board, and producing the paperwork.

### 6.1 Dashboard

The dashboard is the admin landing page and loads in under 1.5 s from cached aggregates (Redis, 5-minute TTL, invalidated on relevant writes).

**KPI cards**

- Total crew, and counts per status: Available, On Board, Sign Off, Medical Leave, Inactive/Blacklisted.
- Active contracts.
- Documents and certificates expiring in 30, 60 and 90 days, plus already expired.
- Crew due for relief in the next 30 days (from Rotation Planning).
- New applicants (status New) and unread contact messages.

**Charts**

- Crew by status (donut).
- Crew by rank, grouped by department (horizontal bar).
- Crew on board by principal (bar).
- Sign-ons and sign-offs per month, last 12 months (line).

**Tables**

- Expiring documents and certificates, sorted by nearest expiry, with color badges and a link to the crew record.
- Upcoming reliefs, sorted by planned relief date.

### 6.2 Master Data

Master data feeds every dropdown in the system. Each master supports create, read, update, soft delete, active/inactive toggle and drag-and-drop sort order. A record in use cannot be hard-deleted.

| Master | Fields |
| --- | --- |
| Company / Principal | Code, name, type (Company, Principal, Both), full address, country, phone, email, website, logo (WebP) |
| Vessel | Name, IMO number (7 digits, unique), flag, vessel type, GRT, HP/kW, year built, owner company, principal, status (Active, Laid Up, Sold) |
| Rank | Name, short code (e.g. CE), department (Deck, Engine, Catering/Other), sort order |
| Vessel Type | Name (AHTS, Crew Boat, ASD Tug, Tanker, Bulk Carrier, RORO, Other) |
| Currency | ISO 4217 code (IDR, USD, EUR, SGD), symbol, decimal places |
| Crew Status | Name, color, system flag (system statuses cannot be deleted) |
| Allowance Type | Name (Overtime, Leave Pay, Standby Pay, Offshore Allowance, Bonus), calculation basis (fixed per month, fixed per day, percentage of basic) |
| Document Type | Name (KTP, Passport, Seaman Book/CDC, Visa, MCU, Yellow Fever), has expiry (yes/no), number format hint |
| Certificate Type | Name, STCW reference (e.g. VI/1), category (COC, COP, Endorsement, Other), has expiry |

**Default ranks (seeded)**

- Deck: Master, Chief Officer, 2nd Officer, 3rd Officer, Bosun, AB, OS.
- Engine: Chief Engineer, 2nd Engineer, 3rd Engineer, 4th Engineer, Engine Cadet, Oiler, Wiper.
- Catering/Other: Cook, Steward.

**Compliance Matrix**

The Compliance Matrix defines which document types and certificate types are mandatory for each rank. It is edited as a grid: ranks as rows, document and certificate types as columns, a checkbox per cell. A crew member is compliant for a rank when every mandatory item exists and is valid (not expired) on the check date.

### 6.3 Crew Management

Crew records are listed in the datagrid (6.9) and edited through a multi-tab form. Each tab saves independently; unsaved changes trigger a leave-page warning.

Every crew member receives a unique 9-character Crew ID on creation (see 11.1). The header of the crew page shows photo, full name, Crew ID, current rank, status badge and compliance badge.

| Tab | Fields and behaviour |
| --- | --- |
| 1. Personal Data | Full name, place of birth, date of birth (age auto-computed), gender, nationality, religion, marital status, height, weight, safety shoe size, coverall size, applied rank, profile photo (auto WebP) |
| 2. Contact & Address | Full address, city, province, country, postal code, home phone, mobile phone (E.164, e.g. +62…), email, nearest airport |
| 3. Bank Information | One or more accounts: bank name, account name, account number, currency, cover bank, remarks (e.g. DIRECT), primary flag |
| 4. Next of Kin | One or more contacts: name, relationship (Spouse, Parent, Child, Sibling, Other), phone, address, primary flag |
| 5. Documents | Rows: document type, number, issued date, expiry date, place of issue, file upload, actions (view, replace, delete). Expiry badge per row |
| 6. Certificates (STCW) | Rows: certificate type, number, issued date, expiry date, issuing authority, place of issue, file upload, actions. Expiry badge per row |
| 7. Experiences | Sea service rows: vessel name, rank, vessel type, GRT/HP, company, principal, sign-on date, sign-off date, duration, status, remarks, source (Manual or Contract). Summary panel shows total sea service per rank |
| 8. Current Assignment | Read-only view of the active contract: vessel, rank, principal, sign-on date, planned relief date, days on board. Empty when not on board |
| 9. Compliance Status | Checklist of mandatory items for the crew's current or applied rank, from the Compliance Matrix: each item marked Valid, Expiring, Expired or Missing, plus an overall Compliant / Not Compliant badge |

**Expiry alert colors** (Documents and Certificates)

| Days to expiry | Badge |
| --- | --- |
| Expired | Dark red |
| 0–30 | Red |
| 31–60 | Orange |
| 61–90 | Yellow |
| Over 90, or no expiry | Green |

**Crew CV upload:** the admin can attach an external CV file in the Personal Data tab. Images are converted to WebP; DOC/DOCX are converted to PDF (see Section 10).

### 6.4 Contract Management

A contract links one crew member to one vessel, rank and principal for a period, with salary and allowances in one currency.

**Contract fields**

- Contract ID (9 characters, auto), crew, vessel, rank, principal, employer company.
- Sign-on date, contract duration (months), planned sign-off date (auto: sign-on + duration, editable), actual sign-off date.
- Point of hire, place of sign-on, place of sign-off.
- Currency (from Master Currency).
- Basic salary (monthly amount).
- Allowances: zero or more rows of allowance type, calculation basis, amount or percentage, computed monthly amount.
- Working hours per week, leave days per month, probation period, notes.
- Status: Draft, Active, Completed, Terminated.

**Contract lifecycle**

1. **Draft:** editable; can generate a draft SEA PDF.
2. **Active:** set when the admin confirms sign-on. The crew status changes to On Board, and Current Assignment shows this contract. A crew member can hold only one Active contract.
3. **Completed:** set when the admin records actual sign-off. The system automatically creates an Experience row from the contract (vessel, rank, vessel type, GRT/HP, company, principal, sign-on, sign-off, status SIGN OFF, source Contract) and sets the crew status to Sign Off.
4. **Terminated:** early end with a mandatory reason. It also creates an Experience row, with remarks holding the reason.

Total monthly pay = basic salary + sum of allowance monthly amounts, shown in the contract currency only (no conversion).

### 6.5 Crew Rotation Planning

Rotation Planning shows who is on board each vessel and when their relief is due, so the admin can line up replacements early.

- **Vessel timeline view:** one lane per vessel, one bar per active contract from sign-on to planned sign-off; bars turn orange 30 days before planned sign-off and red when overdue.
- **Relief plan:** for an active contract, the admin assigns a planned reliever (crew member), planned relief date and joining port. Status: Planned, Confirmed, Completed, Cancelled.
- **Reliever suggestions:** when choosing a reliever, the system lists crew with status Available, the same rank, and Compliant for that rank on the relief date, sorted by total sea service in that rank.
- **Conflict checks:** a reliever cannot be planned on two overlapping reliefs, and a warning shows when any mandatory item expires before the planned contract end.
- Completing a relief creates the reliever's new Draft contract pre-filled from the vessel, rank and principal.

### 6.6 Document Builder

The document builder produces three document kinds from editable templates: Seafarer Employment Agreement (SEA), Quotation, and Crew CV.

**Templates**

- Each template has name, kind (SEA, Quotation, CV), paper size (A4, Letter), orientation, header, body and footer.
- Edited in a rich text editor (TipTap) with tables, images, page breaks and a variable picker.
- One default template per kind; several templates per kind allowed (e.g. per principal).
- The company logo, name and address in header/footer come from Admin Settings.

**Variable tags**

Tags use `{snake_case}` syntax and are replaced with real data at generation time. Unknown tags are flagged in the editor and rendered empty.

| Group | Example tags |
| --- | --- |
| Crew | {crew\_id}, {crew\_name}, {date\_of\_birth}, {place\_of\_birth}, {nationality}, {passport\_no}, {passport\_expiry}, {seaman\_book\_no}, {address}, {mobile\_phone} |
| Contract | {contract\_id}, {vessel\_name}, {imo\_number}, {flag}, {rank}, {principal\_name}, {sign\_on\_date}, {sign\_off\_date}, {duration\_months}, {currency}, {basic\_salary}, {allowance\_table}, {total\_monthly\_pay} |
| Quotation | {quotation\_id}, {client\_name}, {quotation\_date}, {valid\_until}, {pricing\_table}, {subtotal}, {tax}, {grand\_total}, {terms} |
| CV | {photo}, {experience\_table}, {certificate\_table}, {document\_table}, {sea\_service\_summary} |
| Company | {company\_name}, {company\_address}, {company\_phone}, {company\_email}, {today} |

Table tags (`{allowance_table}`, `{experience_table}` and similar) render as formatted tables.

**Quotation**

- Quotation ID (auto), client (principal), vessel (optional), currency, quotation date, valid-until date, status (Draft, Sent, Accepted, Rejected, Expired).
- Pricing mode per quotation: **Daily Rate** (lines of rank, number of crew, daily rate, days, line total) or **Lump Sum** (lines of description, amount), or both sections in one quotation.
- Tax percentage (optional), discount (optional), terms and conditions text.

**Crew CV**

- Generated from crew data in a standard maritime CV layout: photo, personal data, documents, STCW certificates, sea service history and totals per rank.
- The admin can hide sections per generation (e.g. hide bank data, which is never included by default).

**Generation and preview**

- Template HTML is merged with data by the Go API and rendered to PDF by Gotenberg (Chromium). This replaces gofpdf and wkhtmltopdf from the v1.0 draft, because Chromium renders HTML and CSS faithfully.
- Live preview in the browser before export.
- Each generated PDF is compressed (Section 10), stored in S3 with a random name, and logged in a Generated Documents history (kind, reference, template, date, file).

### 6.7 Email Dispatch (Brevo)

Generated contracts, quotations and CVs can be sent by email directly from the system through the Brevo transactional API.

- Send dialog: To (multiple), CC, subject and body pre-filled from an email template, the generated PDF attached; extra attachments optional.
- Email templates are managed in Admin Settings, with variable tags.
- Sending runs in a background job; failures retry 3 times with backoff.
- Each send is logged: recipients, subject, document, Brevo message ID, status (Queued, Sent, Failed), timestamp. The log is visible on the document and on the crew record.
- Brevo is also used for website notifications to the admin (new application, new contact message) and the applicant confirmation email (Part B and C).

### 6.8 Sea Service Calculation

Total sea service is computed per rank and in total from Experiences, merging overlapping periods so days are not double counted. It is shown as years, months and days (e.g. 3 y 4 m 12 d) in the Experiences tab, the CV and the datagrid filter.

### 6.9 Datagrid, Search, Filter, Import/Export

- **Server-side pagination:** page size 10, 20, 30, 50 or 100; default 20.
- **Reactive search:** debounced (300 ms) search on name, Crew ID, passport number and seaman book number, without page reload.
- **Filters:** status, rank, department, vessel type, current vessel, principal, age range, nationality, compliance status, document validity (valid, expiring within N days, expired), minimum sea service in a rank.
- **Grouping:** by principal or company, with group counts.
- **Sort:** ascending or descending on every column.
- **Column chooser:** show or hide columns; the choice is remembered per browser.
- **Export:** the current filtered result to CSV or Excel (.xlsx), generated in a background job for large sets and downloaded via presigned URL.
- **Import:** CSV upload using a downloadable template. The system validates every row first and shows a preview with errors per row; the admin then confirms. Valid rows are inserted in batches; a result report lists created, skipped and failed rows.

The same datagrid component is reused for contracts, quotations, vessels, applicants and master data lists.

## 7. Part B: Website CMS (Admin)

The CMS lets the admin run the public website without code: every page, slide, article, vacancy and contact detail is managed here and served to Next.js through the Public API.

Shared rules for all CMS content:

- Status Draft or Published; only Published content is visible publicly.
- Images are converted to WebP on upload with a responsive set (480, 960, 1600 px wide).
- Every published change triggers on-demand revalidation of the affected Next.js paths, so updates appear within seconds without a rebuild.
- Lists support drag-and-drop ordering and active/inactive toggles where noted.

### 7.1 Pages (Page Builder)

- Fields: title, slug, SEO title, meta description, OG image, status, show in header menu, show in footer menu, menu order.
- **Auto slug:** generated from the title (lowercase, hyphenated, ASCII). Editable; uniqueness is enforced and a suffix (-2, -3) is added on conflict. Changing the slug of a published page creates a 301 redirect from the old slug.
- **Builder:** the page body is an ordered list of blocks, added, edited, reordered (drag and drop), duplicated, hidden or deleted. Stored as JSON (block type plus settings).

| Block | Settings |
| --- | --- |
| Rich Text | Rich text content (headings, lists, links, tables) |
| Image + Text Split | Image, heading, text, button, image left or right |
| Gallery | Selected gallery images, columns (2–4), lightbox on or off |
| Video Embed | YouTube or Vimeo URL, title; loads a lightweight facade until clicked |
| CTA Banner | Heading, text, button label and link, background image or color |
| Stats Counter | Items of number, suffix, label; counts up once when scrolled into view |
| FAQ Accordion | Items of question and answer |
| Card Grid | Cards of icon or image, title, text, link; columns (2–4) |

- System pages Home, News, Career and Contact have fixed layouts but editable SEO fields and intro text.

### 7.2 News

- One category (no category management). Fields: title, slug (auto, editable), excerpt, body (rich text), thumbnail (auto WebP, used as OG image), publish date, status, SEO title, meta description.
- Scheduled publishing: a future publish date keeps the article hidden until that time.

### 7.3 Hero Slider

- Slides: image (WebP, desktop and optional mobile variant), heading, subheading, button label and link, order, active.
- Autoplay interval configurable in Settings (default 6 s); pauses on hover and respects reduced motion.

### 7.4 Gallery

- Images with caption, alt text (required), order, active, and a Show on Home Page flag.
- Bulk upload of multiple images in one action.

### 7.5 Testimonials

- Fields: name, position or company, photo (optional), quote text, rating 1–5 stars, order, active.
- Shown on the home page as a text slider with star ratings.

### 7.6 Client Logos

- Fields: client name, logo (WebP or SVG sanitized), website link (optional), order, active.

### 7.7 Stats

- Home page stats items: number, suffix (e.g. +), label (e.g. Seafarers Deployed), order, active.

### 7.8 Job Vacancies

- Fields: Vacancy ID (9 characters, auto), title, slug (auto, editable), rank, vessel type, department, location or area, employment type (Contract, Permanent), contract duration, salary range text (optional), description, requirements, required documents list, closing date, status (Draft, Open, Closed).
- A vacancy closes automatically after its closing date and disappears from the public list (its page then shows Closed).
- The admin sees an application count per vacancy.

### 7.9 Applicants

- Every public application creates an applicant record with Application ID (9 characters, auto), vacancy, full name, email, phone, date of birth, nationality, rank applied, years of experience, cover message, and uploaded files (CV plus supporting documents).
- **Pipeline:** New, Screening, Interview, Accepted, Rejected. The admin moves applicants between stages in a list or kanban view, with an internal note per stage change.
- Applicants stay a separate archive and are not converted into crew records in v1.
- Search and filter by vacancy, stage, rank, date.
- Files open through presigned URLs only.

### 7.10 Contact Inbox

- Every contact form submission is stored: name, email, phone, subject, message, IP, date, read/unread, archived.
- A notification email is sent to the admin address from Settings.

### 7.11 Languages

- List of languages offered in the public language switcher: name, Google Translate language code (e.g. id, zh-CN, ja), flag (SVG upload or bundled flag set), order, active.
- English is the source language and cannot be removed.

### 7.12 Menus and Footer

- Header menu and footer menu are built from pages, system pages and custom links, with drag-and-drop order.
- Footer text, social media links and office contact come from Settings.

## 8. Part C: Public Website (Next.js)

The public website is a Next.js (App Router) application rendered on the server, with English as the source language and all content, branding and contact details coming from the CMS and Settings.

### 8.1 Pages and routes

| Route | Content |
| --- | --- |
| `/` | Hero slider, intro, stats counter, featured services (card grid), latest 3 news, open vacancies (latest 4), gallery highlights, testimonials slider, client logos, CTA banner |
| `/[slug]` | Dynamic CMS page rendered from builder blocks (About, Services, Terms and Conditions, Privacy Policy, and any new page) |
| `/news` | Paginated article list (9 per page) with thumbnails |
| `/news/[slug]` | Article with thumbnail, date, body, share buttons, 3 related latest articles |
| `/career` | Open vacancies with search and filters |
| `/career/[slug]` | Vacancy detail and application form |
| `/contact` | Office details, embedded map (lazy facade), contact form |
| `/gallery` | Full gallery grid with lightbox |
| `/sitemap.xml`, `/robots.txt` | Generated automatically |
| any unknown path | Custom 404 page with links to Home, Career and Contact |

### 8.2 Career portal

- **Search:** keyword over title, rank and description.
- **Filters:** rank, department, vessel type, location; filters are kept in the URL query so results are shareable and SSR-rendered.
- Each vacancy card shows title, Vacancy ID, rank, vessel type, location and closing date.
- **Application form** (no login): full name, email, phone (E.164), date of birth, nationality, rank applied, years of sea experience, cover message, CV (required, one file), supporting documents (optional, up to 10 files), consent checkbox linking to the Privacy Policy, Cloudflare Turnstile.
- Allowed files: PDF, JPG, JPEG, PNG, WebP, DOC, DOCX, validated by magic bytes and size limit from Settings (Section 10).
- On success the applicant sees the Application ID; an automatic confirmation email with the ID is sent to the applicant, and a notification email with a link to the admin record is sent to the admin.
- Duplicate guard: the same email cannot apply to the same vacancy twice within 30 days.

### 8.3 Contact form

- Fields: name, email, phone (optional), subject, message (max 2,000 characters), Turnstile.
- Stored in the Contact Inbox and emailed to the admin.

### 8.4 Form protection (all public forms)

- Cloudflare Turnstile token verified server-side by the Go API before any processing.
- Rate limit per IP: 5 submissions per 10 minutes per form.
- Hidden honeypot field; submissions that fill it are silently dropped.
- Server-side validation of every field; client-side validation mirrors it for instant feedback.
- Submit button shows a spinner and is disabled while the request runs.

### 8.5 Language switcher

- The header shows the active language as an SVG flag; the dropdown lists active languages from 7.11.
- Translation uses the Google Translate website widget (`translate_element.js`) loaded only after the first switch to a non-English language, so English pages carry no translation script.
- The Google toolbar and branding banner are hidden with CSS; the chosen language is kept in the `googtrans` cookie.
- If the widget fails to load, the site stays fully usable in English. See Risk R1.
- The admin panel is not translated.

### 8.6 Global elements

- Header: logo, main menu, language switcher, Apply Now button linking to Career.
- Footer: company info, footer menu, social links, contact details, copyright with the current year.
- Floating WhatsApp button (number and default message from Settings; hidden when the number is empty).
- Cookie-free analytics is optional and off by default.

### 8.7 Visual direction

Professional maritime style: navy primary (`#0B2545` default, configurable in Settings), white surfaces, one accent color, generous spacing and a clean sans-serif typeface loaded through `next/font` with a system fallback stack.

## 9. Admin Settings Panel

All runtime configuration lives in the Settings panel, grouped in tabs; only infrastructure bootstrap values (database, Redis, encryption key) stay in environment variables.

- Settings are stored as key-value rows in PostgreSQL and cached in Redis; saving a tab invalidates the cache and triggers public site revalidation.
- Secret values (API keys, secret keys, passwords) are encrypted with AES-256-GCM using `APP_ENCRYPTION_KEY`, shown masked, and never returned in full by the API.
- Each integration tab has a **Test Connection** button (send a test email, write and delete a test object, verify a Turnstile key).

| Tab | Settings |
| --- | --- |
| General | Site name, tagline, admin panel title, default language, timezone (default Asia/Jakarta, WIB), date format (default DD-MMM-YYYY), maintenance mode for the public site |
| Company | Legal company name, logo (light and dark), favicon, address, country, phone, email, website, tax number, signatory name and title (used in documents) |
| Contact & Social | Office address, map coordinates, phones, public email, WhatsApp number and default message, social links (LinkedIn, Facebook, Instagram, YouTube, X) |
| Branding | Primary color, accent color, public footer text |
| SEO | Default meta title pattern, default meta description, default OG image, Google Search Console verification code, robots indexing on/off |
| Email (Brevo) | API key (secret), sender name, sender email, admin notification address(es), reply-to address |
| Email Templates | Templates for: Send Contract, Send Quotation, Send CV, Applicant Confirmation, Admin New Application, Admin New Contact Message. Each has subject and HTML body with variable tags and a preview |
| Object Storage | Provider (AWS S3 or S3-compatible such as MinIO), endpoint, region, bucket, access key, secret key (secret), path-style toggle, presigned URL lifetime (default 5 min) |
| Uploads | Max file size per upload (default 10 MB), max files per applicant submission (default 10), WebP quality (default 80), max image dimension (default 2,000 px), PDF compression preset (default ebook, 150 dpi) |
| Security | Cloudflare Turnstile site key and secret key (secret), session idle timeout (default 8 h, read-only display), recovery codes (regenerate) |
| Backup | Enable daily backup, run time (default 02:00 WIB), retention days (default 30), backup bucket/prefix, last backup status, Run Backup Now |
| Alerts | Expiry thresholds in days (default 30, 60, 90) |
| Website | Hero autoplay interval, home sections on/off and order, news per page, career page intro text |
| Account | Admin name, email, change password |

## 10. File Handling Pipeline & Upload Security

Every uploaded file, from the admin panel or a public form, passes through one Go pipeline that validates it by content, normalizes its format, renames it randomly, and stores it privately in S3.

### 10.1 Pipeline steps

1. **Size check:** reject when larger than the limit from Settings, before reading the full body (`http.MaxBytesReader`).
2. **Name check:** reject double extensions (`cv.pdf.exe`), null bytes, path separators and names over 255 characters. The original name is kept only as metadata, never used as a path.
3. **Extension whitelist** per context (table 10.2).
4. **Magic bytes check:** read the first 512 bytes and detect the real type (`h2non/filetype` plus `http.DetectContentType`). The detected type must match both the whitelist and the claimed extension; otherwise reject with `invalid_file_type`.
5. **Deep validation:** images must decode fully (corrupt or polyglot images fail); PDFs must parse and must not contain JavaScript or embedded files (rejected); DOC/DOCX must be valid OLE or ZIP containers, and DOCX with macros (`vbaProject.bin`) is rejected.
6. **Transform** by type (table 10.3).
7. **Rename:** a random lowercase alphanumeric name of 12 characters from crypto/rand, plus the final extension, e.g. `n87hds5300gx.webp`, `7hdt6209bcqk.pdf`.
8. **Store** in S3 under `<entity>/<yyyy>/<mm>/<random-name>` with private ACL and the correct `Content-Type`.
9. **Record** a row in `files`: storage key, original name, mime type, size before and after, checksum (SHA-256), owner entity and ID.

Heavy steps (6) run in a bounded worker pool so many uploads cannot exhaust CPU; the request returns once the file is stored, with a processing status for large conversions.

### 10.2 Allowed types per context

| Context | Allowed input |
| --- | --- |
| Profile photo, CMS images, logos | JPG, JPEG, PNG, WebP (SVG only for logos and flags, sanitized) |
| Crew documents and certificates | PDF, JPG, JPEG, PNG, WebP |
| Crew CV (admin) | PDF, JPG, JPEG, PNG, WebP, DOC, DOCX |
| Applicant CV and documents (public) | PDF, JPG, JPEG, PNG, WebP, DOC, DOCX |
| CSV import | CSV only (UTF-8, validated as text) |

### 10.3 Transformations

| Input | Output | Method |
| --- | --- | --- |
| JPG, JPEG, PNG, WebP | WebP | libvips via `govips`: auto-rotate by EXIF, strip all metadata (including GPS), resize so the longest side is at most the max dimension, encode at quality from Settings (default 80) |
| PDF | PDF | Ghostscript `-dPDFSETTINGS=/ebook` (150 dpi). If the result is not smaller, the original is kept. Text stays selectable |
| DOC, DOCX | PDF | Gotenberg (LibreOffice) conversion, then the PDF compression step |
| SVG | SVG | Sanitized: scripts, event handlers, external references and `foreignObject` removed |

### 10.4 Access

- Buckets are private; no file has a public URL, except published CMS images, which are served through the Next.js image route or a CDN path from a separate public prefix.
- Crew, applicant and generated documents are served only via presigned GET URLs (5 minutes) after the session check.
- Downloads use `Content-Disposition: attachment` for documents and `X-Content-Type-Options: nosniff` everywhere.

## 11. Business Rules

These rules are enforced in the Go service layer and covered by unit tests; the UI only mirrors them.

### 11.1 Unique public IDs

- Crew, contracts, quotations, vacancies, applications and generated documents get a public ID of exactly 9 characters, e.g. `SR6H70QU9`.
- Alphabet: uppercase letters and digits without look-alikes (no 0, O, 1, I, L): 31 characters, about 2.6 × 10^13 combinations.
- Generated with crypto/rand; a unique index per table guards collisions, and the service retries up to 5 times on conflict.
- Public IDs are immutable and appear in documents, emails and URLs. Internal primary keys are UUID v7 and never shown to users.

### 11.2 Crew status transitions

| Event | New status |
| --- | --- |
| Crew created | Available |
| Contract set to Active | On Board |
| Contract Completed or Terminated | Sign Off |
| Admin marks medical leave | Medical Leave |
| Admin marks ready | Available |
| Admin blacklists (reason required) | Inactive/Blacklisted |

- Only Available crew can be assigned to a new Active contract or planned as a reliever.
- Blacklisted crew are excluded from reliever suggestions and exports by default.

### 11.3 Expiry and compliance

- Expiry badge thresholds come from Settings (default 30, 60, 90 days), computed against today in WIB.
- A document or certificate type without expiry is always Valid once present.
- Compliance for a rank = all mandatory items in the Compliance Matrix exist and are Valid on the check date. Expiring items still count as Valid but are flagged.
- Compliance is checked against: today (crew page, datagrid), the planned relief date (reliever suggestions), and the planned contract end (warnings).

### 11.4 Contracts

- One Active contract per crew member at a time.
- Sign-off date must be on or after sign-on date.
- Completing or terminating a contract creates exactly one Experience row (idempotent: re-saving does not duplicate it).
- Allowance by percentage = basic salary × percentage; by day = daily amount × 30.

### 11.5 Sea service

- Duration per row = sign-off date − sign-on date + 1 day; an open row (on board) counts to today.
- Totals per rank merge overlapping periods before summing.
- Display format: years, months, days, using 30-day months and 365-day years for totals.

### 11.6 Money and currency

- Amounts are stored as `NUMERIC(18,2)` plus a currency code; never as floats.
- No currency conversion. Any sum across records (dashboard, reports, exports) is grouped by currency, e.g. USD 45,000.00 | IDR 120,000,000.
- Display format follows the currency's decimal places from Master Currency.

### 11.7 Dates and time

- Timestamps are stored in UTC (`timestamptz`); date-only fields as `date`.
- Display uses the timezone and format from Settings: default Asia/Jakarta (WIB) and `DD-MMM-YYYY`, e.g. 20-Oct-2026; with time, `DD-MMM-YYYY HH:mm`.

### 11.8 Slugs

- Generated from the title: transliterate to ASCII, lowercase, non-alphanumerics to hyphens, collapse repeats, trim, max 80 characters.
- Editable by the admin with the same normalization applied; unique per content type.
- Changing a published slug stores a 301 redirect from the old path.

### 11.9 Deletion

- Crew, contracts, quotations, vacancies, applicants and CMS content use soft delete (`deleted_at`).
- Files linked to soft-deleted records are kept; a monthly job purges files of records deleted more than 90 days ago.

## 12. Data Model

PostgreSQL 17 holds about 45 tables in four groups; every table has `id UUID v7`, `created_at`, `updated_at`, and business tables add `deleted_at` and a 9-character `public_id` where listed.

Conventions: snake\_case plural table names, foreign keys as `<entity>_id` with indexes, money as `NUMERIC(18,2)` plus `currency_code`, enums as `TEXT` with CHECK constraints, migrations managed by `golang-migrate`.

### 12.1 System and settings

| Table | Key columns |
| --- | --- |
| admins | name, email (unique), password\_hash, password\_changed\_at, must\_change\_password |
| recovery\_codes | admin\_id, code\_hash, used\_at |
| settings | group, key (unique per group), value (JSONB), is\_secret (value encrypted) |
| files | storage\_key, original\_name, mime\_type, size\_original, size\_stored, sha256, owner\_type, owner\_id, status |
| email\_templates | code (unique), subject, body\_html |
| email\_logs | template\_code, to, cc, subject, related\_type, related\_id, brevo\_message\_id, status, error, sent\_at |
| backups | storage\_key, size, status, started\_at, finished\_at, error |
| redirects | from\_path (unique), to\_path, status\_code |

### 12.2 Master data

| Table | Key columns |
| --- | --- |
| companies | code, name, type, address, country, phone, email, website, logo\_file\_id, is\_active, sort\_order |
| vessels | name, imo\_number (unique), flag, vessel\_type\_id, grt, hp\_kw, year\_built, owner\_company\_id, principal\_id, status |
| ranks | name, short\_code, department, sort\_order, is\_active |
| vessel\_types | name, sort\_order, is\_active |
| currencies | code (ISO 4217, unique), symbol, decimals, is\_active |
| crew\_statuses | name, color, is\_system |
| allowance\_types | name, basis, is\_active |
| document\_types | name, has\_expiry, number\_hint, is\_active |
| certificate\_types | name, stcw\_reference, category, has\_expiry, is\_active |
| compliance\_requirements | rank\_id, requirement\_type (document or certificate), document\_type\_id, certificate\_type\_id; unique per rank and type |

### 12.3 Crew operations

| Table | Key columns |
| --- | --- |
| crews | public\_id, full\_name, place\_of\_birth, date\_of\_birth, gender, nationality, religion, marital\_status, height\_cm, weight\_kg, shoe\_size, coverall\_size, applied\_rank\_id, crew\_status\_id, photo\_file\_id, cv\_file\_id, address fields, home\_phone, mobile\_phone, email, nearest\_airport |
| crew\_bank\_accounts | crew\_id, bank\_name, account\_name, account\_number, currency\_code, cover\_bank, remarks, is\_primary |
| crew\_next\_of\_kins | crew\_id, name, relationship, phone, address, is\_primary |
| travel\_documents | crew\_id, document\_type\_id, number, issued\_date, expired\_date, place\_of\_issue, file\_id |
| certificates | crew\_id, certificate\_type\_id, number, issued\_date, expired\_date, issuing\_authority, place\_of\_issue, file\_id |
| experiences | crew\_id, vessel\_name, vessel\_id (nullable), rank\_id, vessel\_type\_id, grt\_hp, company\_id, principal\_id, sign\_on\_date, sign\_off\_date, status, remarks, source, contract\_id (nullable, unique) |
| contracts | public\_id, crew\_id, vessel\_id, rank\_id, principal\_id, employer\_company\_id, sign\_on\_date, duration\_months, planned\_sign\_off\_date, actual\_sign\_off\_date, point\_of\_hire, sign\_on\_port, sign\_off\_port, currency\_code, basic\_salary, working\_hours\_week, leave\_days\_month, probation\_months, status, termination\_reason, notes |
| contract\_allowances | contract\_id, allowance\_type\_id, basis, value, monthly\_amount |
| relief\_plans | contract\_id, reliever\_crew\_id, planned\_relief\_date, joining\_port, status, notes |
| document\_templates | name, kind, paper\_size, orientation, header\_html, body\_html, footer\_html, is\_default |
| quotations | public\_id, client\_company\_id, vessel\_id, currency\_code, quotation\_date, valid\_until, tax\_percent, discount\_amount, terms, status, subtotal, grand\_total |
| quotation\_lines | quotation\_id, line\_type (daily\_rate or lump\_sum), rank\_id, description, crew\_count, daily\_rate, days, amount, sort\_order |
| generated\_documents | public\_id, kind, template\_id, related\_type, related\_id, file\_id |

### 12.4 Website CMS

| Table | Key columns |
| --- | --- |
| pages | title, slug (unique), blocks (JSONB), seo\_title, meta\_description, og\_image\_file\_id, status, in\_header\_menu, in\_footer\_menu, menu\_order, is\_system |
| news | title, slug (unique), excerpt, body\_html, thumbnail\_file\_id, published\_at, status, seo\_title, meta\_description |
| hero\_slides | image\_file\_id, mobile\_image\_file\_id, heading, subheading, button\_label, button\_link, sort\_order, is\_active |
| gallery\_images | image\_file\_id, caption, alt\_text, show\_on\_home, sort\_order, is\_active |
| testimonials | name, position, photo\_file\_id, quote, rating (1–5), sort\_order, is\_active |
| client\_logos | name, logo\_file\_id, url, sort\_order, is\_active |
| stats\_items | number, suffix, label, sort\_order, is\_active |
| menu\_items | menu (header or footer), label, page\_id or url, parent\_id, sort\_order |
| vacancies | public\_id, title, slug (unique), rank\_id, vessel\_type\_id, department, location, employment\_type, duration, salary\_text, description\_html, requirements\_html, required\_documents, closing\_date, status |
| applicants | public\_id, vacancy\_id, full\_name, email, phone, date\_of\_birth, nationality, rank\_applied\_id, years\_experience, message, stage, consent\_at, ip\_address |
| applicant\_files | applicant\_id, file\_id, kind (cv or document) |
| applicant\_stage\_logs | applicant\_id, from\_stage, to\_stage, note, created\_at |
| contact\_messages | name, email, phone, subject, message, ip\_address, is\_read, is\_archived |
| languages | name, code (unique), flag\_file\_id, sort\_order, is\_active, is\_source |

### 12.5 Key indexes

- Trigram (`pg_trgm`) GIN indexes on `crews.full_name`, passport and seaman book numbers for fast fuzzy search.
- Composite indexes for the datagrid's common filters: `(crew_status_id, applied_rank_id)`, `(expired_date)` on documents and certificates.
- Partial unique index: one Active contract per crew (`WHERE status = 'active'`).

## 13. API Endpoint Overview

The Go API exposes two JSON namespaces: `/api/admin/v1` (session required) and `/api/public/v1` (no session, read-only content plus protected forms).

**Conventions**

- REST with plural resources; list endpoints accept `page`, `per_page`, `sort`, `order`, `q` and filter params.
- Response envelope: `{ "data": …, "meta": { "page", "per_page", "total" } }`; errors: `{ "error": { "code", "message", "fields": {…} } }` with correct HTTP status.
- Resources are addressed by `public_id` where one exists, otherwise by UUID.
- Long jobs (export, import, backup, bulk conversion) return `202` with a job ID polled at `/jobs/{id}`.

### 13.1 Admin API (`/api/admin/v1`)

| Area | Endpoints |
| --- | --- |
| Auth | POST /auth/login, POST /auth/logout, GET /me, POST /auth/recovery, PUT /me/password |
| Dashboard | GET /dashboard/summary, GET /dashboard/charts, GET /dashboard/expiring, GET /dashboard/reliefs |
| Master data | CRUD /companies, /vessels, /ranks, /vessel-types, /currencies, /crew-statuses, /allowance-types, /document-types, /certificate-types; PATCH /{master}/reorder |
| Compliance | GET, PUT /compliance-matrix |
| Crew | CRUD /crews; GET /crews/{id}/compliance; GET /crews/{id}/sea-service; PATCH /crews/{id}/status |
| Crew sub-resources | CRUD /crews/{id}/bank-accounts, /next-of-kins, /documents, /certificates, /experiences |
| Contracts | CRUD /contracts; POST /contracts/{id}/activate, /complete, /terminate |
| Rotation | GET /rotation/timeline; CRUD /relief-plans; GET /relief-plans/suggestions?contract\_id=&date=; POST /relief-plans/{id}/complete |
| Templates and documents | CRUD /document-templates; POST /documents/preview; POST /documents/generate; GET /generated-documents |
| Quotations | CRUD /quotations; PATCH /quotations/{id}/status |
| Email | POST /emails/send; GET /email-logs |
| Import/Export | GET /crews/import-template; POST /crews/import/validate; POST /crews/import/commit; POST /crews/export |
| Files | POST /files (multipart); GET /files/{id}/url (presigned); DELETE /files/{id} |
| CMS | CRUD /pages, /news, /hero-slides, /gallery, /testimonials, /client-logos, /stats, /menus, /languages; PATCH reorder on each |
| Career | CRUD /vacancies; GET /applicants; GET /applicants/{id}; PATCH /applicants/{id}/stage |
| Contact | GET /contact-messages; PATCH /contact-messages/{id} |
| Settings | GET /settings/{group}; PUT /settings/{group}; POST /settings/{group}/test; POST /recovery-codes/regenerate |
| Backup | GET /backups; POST /backups/run |
| Jobs | GET /jobs/{id} |

### 13.2 Public API (`/api/public/v1`)

| Endpoint | Purpose |
| --- | --- |
| GET /site | Site name, branding, contact, social, menus, languages, SEO defaults |
| GET /home | All home page sections in one call |
| GET /pages/{slug} | Published page with blocks |
| GET /news, GET /news/{slug} | Article list and detail |
| GET /gallery | Active gallery images |
| GET /vacancies, GET /vacancies/{slug} | Open vacancies with search and filters, vacancy detail |
| GET /vacancy-filters | Ranks, departments, vessel types, locations in use |
| POST /applications | Multipart application with Turnstile token |
| POST /contact | Contact message with Turnstile token |
| GET /redirects/{path} | Resolve legacy slugs |
| GET /sitemap-data | All published URLs with last modified dates |

Public GET responses send `Cache-Control` and `ETag` headers and are cached in Redis; Next.js caches them with tag-based revalidation. Admin writes call the Next.js revalidation endpoint with a shared secret.

## 14. Non-Functional Requirements

The system must be secure by default, fast under 50,000+ crew rows, and score 85 or higher on Google PageSpeed Insights for every public page without cutting any data fetching.

### 14.1 Security

- TLS everywhere (reverse proxy terminates HTTPS; HSTS enabled).
- Security headers on all responses: Content-Security-Policy (allowing Turnstile, Google Translate and video hosts explicitly), X-Content-Type-Options, X-Frame-Options DENY on admin, Referrer-Policy strict-origin-when-cross-origin, Permissions-Policy.
- Parameterized SQL only (sqlc or pgx with placeholders); no string-built queries.
- Rich text and builder HTML sanitized server-side (bluemonday) before storage and rendering.
- CORS: admin API only from the admin origin; public API only from the public site origin.
- Secrets in Settings encrypted at rest (AES-256-GCM); environment secrets never committed.
- Rate limiting in Redis: login, public forms, and a global per-IP limit on the public API.
- Upload security per Section 10.
- Personal data (passport, bank) never appears in logs; request logs mask these fields.

### 14.2 Performance budget

| Metric | Target |
| --- | --- |
| PageSpeed Insights, mobile and desktop, every public page | 85 or higher (green) |
| Largest Contentful Paint (mobile, 4G) | 2.5 s or less |
| Cumulative Layout Shift | 0.1 or less |
| Interaction to Next Paint | 200 ms or less |
| Public JS per page (gzipped, first load) | 150 KB or less |
| Admin datagrid page load, 50,000 crew | 1.5 s or less |
| Admin API p95 for list endpoints | 300 ms or less |
| Public API p95 (cache hit) | 50 ms or less |
| PDF generation (contract, CV) | 10 s or less |

The PageSpeed target is a release gate: Lighthouse CI runs on Home, a CMS page, News detail, Career and Contact, and fails the build below 85.

### 14.3 Caching and concurrency

- **Redis:** settings, site data, home data, master data lists and dashboard aggregates cached with explicit invalidation on write; public GETs cached per URL with short TTL (60 s) plus tag invalidation.
- **Next.js:** server components fetch with cache tags; admin writes trigger `revalidateTag`. Static assets are fingerprinted and served with `Cache-Control: public, max-age=31536000, immutable`.
- **Images:** WebP with width variants, `srcset`/`sizes`, explicit width and height, lazy loading below the fold, `fetchpriority="high"` on the first hero slide only.
- **Go concurrency:** independent queries for one response (e.g. home page sections, dashboard widgets) run in parallel with `errgroup` and a shared context. CPU-heavy work (image, PDF, conversion, export, import, backup) runs in bounded worker pools sized from CPU count; jobs are queued in Redis so the API stays responsive.
- **Database:** pgx connection pool tuned to CPU and Postgres limits, prepared statements, keyset or indexed offset pagination, `COUNT` estimates avoided by cached totals for unfiltered lists.
- Every request carries a context with timeout; cancelled requests stop their queries.

### 14.4 SEO

- Server-rendered HTML for every public route.
- Per page: title, meta description, canonical URL, Open Graph (title, description, image 1200×630, type, url, site\_name), Twitter Card (summary\_large\_image).
- OG image: the page or article's own image, else the default OG image from Settings.
- `hreflang` is not emitted, because translation is client-side.
- `robots.txt` from Settings (indexing on/off) with sitemap reference; `sitemap.xml` generated from all published pages, news and open vacancies with `lastmod`.
- JSON-LD: Organization (site-wide), NewsArticle (news detail), JobPosting (vacancy detail, including `validThrough` from closing date), BreadcrumbList.
- 301 redirects for changed slugs; custom 404 returns HTTP 404.
- Semantic HTML: one `h1` per page, alt text required on images, descriptive link text.

### 14.5 Loading UX rules

- Content is rendered on the server with its data; pages never ship an empty shell that fills in later.
- No skeleton or placeholder is used to fake speed or to hide data that could have been rendered.
- A spinner appears only for genuinely slow user actions: form submit, file upload, PDF generation, export, and route transitions that exceed 300 ms (via `loading.tsx` delayed by 300 ms).
- Data fetching is never reduced, truncated, deferred or skipped to improve a score. Every page fetches all the data it needs; speed comes from caching, parallel queries and payload size, not from fetching less.
- Layout space for images and embeds is reserved to prevent layout shift.

### 14.6 Animation rules

- CSS transitions and transforms only (`transform`, `opacity`, `grid-template-rows` for accordion height); no JS animation libraries on the public site.
- Durations 150–300 ms with ease-out; no parallax, no scroll-jacking, no continuous decorative motion.
- FAQ accordion: smooth height transition with `grid-template-rows: 0fr → 1fr`, chevron rotation, keyboard accessible (`button`, `aria-expanded`).
- Sliders: CSS scroll-snap or a lightweight component under 10 KB; autoplay pauses on hover, focus and hidden tab.
- Stats counter animates once when visible (IntersectionObserver).
- All motion is disabled under `prefers-reduced-motion: reduce`.

### 14.7 Availability and backup

- Daily `pg_dump` (custom format, compressed) at the time set in Settings, uploaded to S3, retained for the configured days (default 30); failures shown on the dashboard.
- Restore procedure documented in the repository (`docs/restore.md`) and tested before launch.
- Health endpoints: `/healthz` (process) and `/readyz` (database, Redis, S3 reachable).
- Structured JSON logs (slog) with request ID.

### 14.8 UI/UX

- Admin: desktop-first, responsive down to 1024 px width, usable on tablet; navy and white maritime theme; dense but tidy tables; sidebar navigation grouped by Crew Operations, Website, Settings.
- Public: mobile-first, responsive from 360 px.
- Accessibility: WCAG 2.1 AA contrast, visible focus states, full keyboard navigation, form errors announced to screen readers.
- Dates shown as DD-MMM-YYYY (WIB), money with currency code.

## 15. Deployment & DevOps

The whole system runs from one `docker-compose.yml` on any Linux host with Docker, with external S3 swappable for the bundled MinIO.

### 15.1 Services

| Service | Image / build | Role | Exposed |
| --- | --- | --- | --- |
| proxy | Caddy 2 | HTTPS (automatic certificates), routing, compression (zstd, gzip), security headers | 80, 443 |
| api | Multi-stage Go build on a slim Debian runtime with libvips and Ghostscript, non-root user | Admin API, Public API, workers, scheduler, CLI | internal |
| admin | React (Vite) build served as static files by Caddy | Admin SPA at `admin.<domain>` | via proxy |
| web | Next.js standalone build | Public website at `<domain>` | via proxy |
| postgres | PostgreSQL 17 | Database, volume `pgdata` | internal |
| redis | Redis 7 | Sessions, cache, rate limits, job queue (AOF on) | internal |
| minio | MinIO (optional profile) | S3-compatible storage when no cloud S3 is used | internal / console optional |
| gotenberg | Gotenberg 8 | HTML to PDF (Chromium), DOC/DOCX to PDF (LibreOffice) | internal |

All internal services live on a private Docker network; only the proxy publishes ports. Each service has a healthcheck and `restart: unless-stopped`.

### 15.2 Environment variables

Only bootstrap and infrastructure values are environment variables; everything else is in Settings.

| Variable | Purpose |
| --- | --- |
| APP\_ENV | production, staging, local |
| APP\_URL, ADMIN\_URL, API\_URL | Public origins used for CORS, links and cookies |
| APP\_ENCRYPTION\_KEY | 32-byte key for encrypting secret settings |
| DATABASE\_URL | PostgreSQL connection string |
| REDIS\_URL | Redis connection string |
| GOTENBERG\_URL | Internal Gotenberg URL |
| REVALIDATE\_SECRET | Shared secret between API and Next.js |
| S3\_BOOTSTRAP\_\* | Optional initial storage values, copied into Settings on first boot |
| LOG\_LEVEL | debug, info, warn, error |

A `.env.example` lists all variables with safe placeholders.

### 15.3 Scheduled jobs (in the api service)

| Job | Schedule |
| --- | --- |
| Database backup to S3 and retention cleanup | Daily, time from Settings (default 02:00 WIB) |
| Close vacancies past closing date | Hourly |
| Expire quotations past valid-until date | Daily |
| Publish scheduled news | Every minute |
| Purge files of records deleted over 90 days ago | Monthly |
| Refresh dashboard aggregates | Every 5 minutes and on write |

### 15.4 CLI commands

- `seafarers migrate up|down`
- `seafarers db:seed`
- `seafarers admin:create`
- `seafarers admin:reset-password --email=<email>`
- `seafarers backup:run`

### 15.5 CI pipeline

On every push: `gofmt` check, `golangci-lint`, `go test ./...`, ESLint and TypeScript check for both frontends, Vitest, Docker image builds, and Lighthouse CI on the public site (gate: 85).

## 16. Seed Data Specification

`seafarers db:seed` loads two sets: **reference** data (always, idempotent) and **demo** data (only when `--demo` is passed, never in production by default).

### 16.1 Reference seed

| Data | Content |
| --- | --- |
| Ranks | 16 ranks across Deck, Engine and Catering/Other (Section 6.2) with short codes |
| Vessel types | AHTS, Crew Boat, ASD Tug, Tanker, Bulk Carrier, RORO, Other |
| Currencies | IDR (0 decimals), USD, EUR, SGD (2 decimals) |
| Crew statuses | Available, On Board, Sign Off, Medical Leave, Inactive/Blacklisted (system flags) |
| Allowance types | Overtime, Leave Pay, Standby Pay, Offshore Allowance, Bonus |
| Document types | KTP, Passport, Seaman Book (CDC), Buku Pelaut, Visa, Medical Check Up, Yellow Fever |
| Certificate types | COC (Master, CO, OOW Deck, CE, 2E, OOW Engine), Basic Safety Training (VI/1), Advanced Fire Fighting (VI/3), Medical First Aid (VI/4), Proficiency in Survival Craft (VI/2), Security Awareness (VI/6), Rating Forming Part of Navigational Watch (II/4), Rating Forming Part of Engineering Watch (III/4), GMDSS, BOSIET, HUET |
| Compliance matrix | Sensible defaults: all ranks require Passport, Seaman Book, MCU, BST, SAT; officers add their COC; offshore vessels add BOSIET/HUET |
| Languages | English (source), Indonesian, with SVG flags |
| Email templates | All six templates from Section 9 with professional default text |
| Document templates | One default SEA (MLC 2006-style clauses), one Quotation, one Crew CV |
| Settings | Defaults from Section 9 with placeholder company data |
| System pages | Home, News, Career, Contact, Gallery |

### 16.2 Legal and company pages (reference seed)

Written for a crew manning agency and editable after install:

- **About Us:** company story, mission, vision, values; uses Image + Text Split, Stats Counter and Card Grid blocks.
- **Services:** crew manning, crew management, recruitment and selection, document and certification handling, travel and logistics; Card Grid and CTA Banner.
- **Terms and Conditions:** use of website, career applications (accuracy of information, no fees charged to applicants), intellectual property, links, limitation of liability, governing law (Republic of Indonesia), changes, contact.
- **Privacy Policy:** data controller, data collected through contact and application forms, purposes (recruitment, communication), legal basis and consent, retention (applicant data kept 24 months), sharing with principals only for recruitment, storage security, data subject rights under Indonesia's Personal Data Protection Law (UU PDP No. 27/2022), cookies (language preference, Turnstile), contact for privacy requests.
- **FAQ:** application process, required documents, fees (none), contract length; uses FAQ Accordion.

### 16.3 Demo seed (`--demo`)

- 5 companies/principals, 10 vessels with valid-format IMO numbers, 60 crew with realistic Indonesian and international names, documents and certificates with spread expiry dates (some expired, some within 30/60/90 days), 120 experience rows, 15 active contracts with allowances, 5 relief plans, 3 quotations.
- 3 hero slides, 12 gallery images, 6 testimonials (ratings 4–5), 8 client logos, 4 stats items, 6 news articles, 6 vacancies (4 open, 2 closed), 15 applicants across all stages, 5 contact messages.
- Demo images are bundled placeholder photos with permissive licenses, processed through the normal WebP pipeline.

## 17. Acceptance Criteria

A module is done when every criterion below passes on a staging deployment with the demo seed.

**Authentication**

- [ ] Any admin URL or admin API call without a session returns the login page or 401; no data is leaked in the response.
- [ ] Session expires after 8 hours idle; logout and password change invalidate the session in Redis immediately.
- [ ] The 6th failed login within 15 minutes from one IP is blocked.
- [ ] CLI reset and a recovery code each restore access; a used recovery code fails a second time.

**Crew and compliance**

- [ ] A new crew member receives a 9-character ID from the allowed alphabet.
- [ ] Documents and certificates show the correct badge color for expired, 0–30, 31–60, 61–90 and 90+ days.
- [ ] Compliance Status turns Not Compliant when a mandatory certificate for the rank expires.
- [ ] Sea service totals per rank do not double count overlapping periods.

**Contracts and rotation**

- [ ] Activating a contract sets the crew to On Board; a second Active contract for the same crew is rejected.
- [ ] Completing a contract creates exactly one Experience row and sets the crew to Sign Off.
- [ ] Totals are shown per currency without conversion.
- [ ] Reliever suggestions list only Available, same-rank crew compliant on the relief date.

**Documents and email**

- [ ] SEA, Quotation (daily rate, lump sum, both) and CV generate a PDF in 10 s or less with all tags replaced and company header/footer.
- [ ] Sending a document via Brevo delivers the email with the PDF attached and logs the Brevo message ID.

**Datagrid**

- [ ] With 50,000 crew, the list loads in 1.5 s or less and search returns in under 500 ms.
- [ ] Export returns exactly the filtered rows; import rejects invalid rows with per-row errors and commits valid ones.

**Files**

- [ ] A renamed executable (`virus.exe` saved as `cv.pdf`) is rejected.
- [ ] A JPG upload is stored as WebP with a random 12-character name and no EXIF data.
- [ ] A DOCX upload is stored as PDF; a DOCX with macros is rejected.
- [ ] A file URL copied from the admin stops working after 5 minutes and never works without a session.

**Website CMS and public site**

- [ ] Publishing a page, article or slide appears on the public site within 10 seconds without a rebuild.
- [ ] Changing a published slug redirects the old URL with 301.
- [ ] Career search and filters work via URL parameters and render server-side.
- [ ] An application without a valid Turnstile token is rejected; a valid one stores the applicant, emails the applicant and the admin, and shows the Application ID.
- [ ] A vacancy past its closing date disappears from the list.
- [ ] Language switch to Indonesian translates the page and keeps the choice across pages; when the Google script is blocked, the English site still works.
- [ ] Home, CMS page, News detail, Career and Contact score 85 or higher on PageSpeed Insights mobile and desktop.
- [ ] Every public page has title, description, canonical, OG and Twitter tags; sitemap.xml and robots.txt are valid; JobPosting JSON-LD passes Google's Rich Results Test.

**Operations**

- [ ] `docker compose up -d` on a clean host brings all services healthy.
- [ ] Daily backup appears in S3; a restore from it succeeds on a fresh database.

## 18. Testing Strategy

Unit tests are mandatory for business logic: Go `testing` with `testify` on the backend and Vitest on both frontends, all run in CI on every push.

| Layer | Tooling | What is tested | Target |
| --- | --- | --- | --- |
| Go services | testing, testify, mocks via interfaces | Business rules in Section 11: IDs, status transitions, compliance, sea service merging, contract lifecycle, money per currency, slugs | 80% line coverage of `internal/service` |
| Go file pipeline | testing, testify, fixture files | Magic bytes rejection, polyglot and renamed files, WebP output, PDF compression fallback, DOCX macro rejection | All rows of tables 10.2 and 10.3 |
| Go handlers and requests | testing, httptest | Validation errors, auth middleware, CSRF, rate limits, response envelope | Every endpoint group |
| Go repositories | testing with a disposable PostgreSQL (testcontainers-go) | Queries, filters, pagination, partial unique indexes | Critical queries |
| React admin | Vitest, Testing Library | Datagrid state, form validation, tab save flow, variable picker | Key components |
| Next.js public | Vitest, Testing Library | Block renderers, career filters, form validation, SEO metadata builders | All builder blocks |
| Performance | Lighthouse CI | PageSpeed budget from 14.2 | Gate at 85 |

Test files are the only place where explanatory comments are allowed (see Appendix A). Tests are table-driven in Go and named after the behaviour they check.

## 19. Development Phases & Milestones

The build runs in six sequential phases over an estimated 14 weeks for a small team (one backend, one frontend developer); crew operations ship before the website so data entry can start early.

&#91;embedded content: Development phases · 6 phases, estimated 14 weeks\]

Phase 2 ends with a usable internal tool, so the admin can begin migrating crew data by CSV import while later phases are built.

## 20. Risks & Mitigations

The largest risk is the free Google Translate widget, which Google no longer officially supports for new sites.

| ID | Risk | Impact | Mitigation |
| --- | --- | --- | --- |
| R1 | Google Translate website widget stops working or changes its markup | Language switcher breaks | Load it lazily and isolate it in one component; the English site stays fully functional; the roadmap holds a paid translation API as replacement |
| R2 | Translation widget and third-party scripts lower PageSpeed score | Score below 85 | Load the widget only after a non-English language is chosen; Turnstile only on form pages; video embeds as click-to-load facades |
| R3 | Large PDF or DOCX conversions overload the server | Slow API | Bounded worker pools, queue in Redis, per-file size limit, Gotenberg timeouts |
| R4 | Single admin account is lost or compromised | Lockout or data exposure | CLI reset and recovery codes; Argon2id; session revocation; rate limits; 2FA on the roadmap |
| R5 | Personal data breach (passports, bank data) | Legal exposure under UU PDP | Private buckets, presigned URLs, encryption in transit, masked logs, backups in private bucket, least-privilege S3 keys |
| R6 | Spam or abuse of public forms | Inbox flooding, storage use | Turnstile, honeypot, per-IP rate limits, file count and size limits, 30-day duplicate guard |
| R7 | Brevo outage or quota exhaustion | Emails not delivered | Retries with backoff, email log with Failed status and Resend action |
| R8 | Data quality in initial CSV import | Wrong crew records | Validate-then-commit import with per-row error report |
| R9 | Backup silently failing | Data loss | Backup status on dashboard, failed backup highlighted, documented and tested restore |

## 21. Glossary

| Term | Meaning |
| --- | --- |
| AHTS | Anchor Handling Tug Supply vessel |
| ASD Tug | Azimuth Stern Drive tug |
| BOSIET | Basic Offshore Safety Induction and Emergency Training |
| BST | Basic Safety Training (STCW VI/1) |
| CDC | Continuous Discharge Certificate, also called Seaman Book |
| COC | Certificate of Competency, the license for an officer rank |
| COP | Certificate of Proficiency, for specific skills or ratings |
| Compliance Matrix | Admin-defined list of mandatory documents and certificates per rank |
| GRT | Gross Register Tonnage |
| HP | Horsepower of the main engine |
| HUET | Helicopter Underwater Escape Training |
| IMO number | Unique 7-digit vessel identifier issued under the International Maritime Organization scheme |
| MCU | Medical Check Up |
| MLC 2006 | Maritime Labour Convention, basis for SEA clauses |
| Principal | Ship owner or operator that hires crew through the manning agency |
| Relief | Replacement of an on-board crew member by another at contract end |
| SAT | Security Awareness Training (STCW VI/6) |
| SEA | Seafarer Employment Agreement |
| Sign on / Sign off | Joining or leaving a vessel under a contract |
| STCW | International Convention on Standards of Training, Certification and Watchkeeping for Seafarers |
| UU PDP | Indonesia's Personal Data Protection Law No. 27 of 2022 |
| WIB | Western Indonesia Time, UTC+7 (Asia/Jakarta) |

## Appendix A: Engineering Guidelines (AGENTS.md)

These rules apply to every human developer and AI coding agent working on the repository; the same text ships as `AGENTS.md` at the repository root.

### A.1 Repository layout

```
seafarers-crew/
├── AGENTS.md
├── docker-compose.yml
├── .env.example
├── backend/      Go API, workers, CLI
├── admin/        React admin SPA (Vite + TypeScript)
├── web/          Next.js public website (TypeScript)
├── deploy/       Caddyfile, Dockerfiles
└── docs/         restore.md, architecture notes
```

### A.2 Backend structure (MVC-style, idiomatic Go)

Layers follow Laravel's mental model while keeping Go conventions: `internal/` for private code, small interfaces defined by the consumer, explicit dependency injection, no global state.

```
backend/
├── cmd/seafarers/main.go        entry point: serve, migrate, db:seed, admin:*, backup:run
├── internal/
│   ├── app/                     bootstrap and dependency wiring
│   ├── config/                  env loading and typed config
│   ├── routes/                  admin.go, public.go (route registration)
│   ├── middleware/              auth, csrf, ratelimit, cors, requestid, recover
│   ├── handler/                 Controllers
│   │   ├── admin/               crew_handler.go, contract_handler.go …
│   │   └── public/              page_handler.go, application_handler.go …
│   ├── request/                 FormRequests: input structs + validation
│   ├── resource/                API Resources: response structs + mappers
│   ├── service/                 business logic (one file per domain)
│   ├── repository/              data access (pgx / sqlc)
│   ├── model/                   domain structs (Eloquent model equivalent, no ORM magic)
│   ├── job/                     workers, queue, scheduler
│   ├── platform/                adapters: postgres, redis, s3, brevo, gotenberg, turnstile, imaging, pdf
│   └── support/                 publicid, slug, money, datetime, errors
├── database/
│   ├── migrations/              golang-migrate SQL files
│   └── seeders/                 reference and demo seeders
└── testdata/                    fixture files for tests
```

| Laravel | Seafarers Crew (Go) |
| --- | --- |
| routes/api.php | internal/routes |
| app/Http/Controllers | internal/handler |
| app/Http/Requests | internal/request |
| app/Http/Resources | internal/resource |
| app/Services | internal/service |
| Repositories / Eloquent queries | internal/repository |
| app/Models | internal/model |
| app/Http/Middleware | internal/middleware |
| app/Jobs, Console/Kernel | internal/job |
| database/migrations, seeders | database/migrations, database/seeders |
| artisan | cmd/seafarers |

Flow: route → middleware → handler (parse request, call service, return resource) → service (business rules, transactions) → repository (SQL). Handlers hold no business logic; repositories hold no business rules.

### A.3 Admin and web structure

```
admin/src/
├── app/          router, providers, layout
├── pages/        one folder per module (crews, contracts, rotation, cms, settings …)
├── components/   ui, datagrid, forms, editor
├── api/          typed API client and endpoint functions
├── hooks/
├── lib/          formatters (date, money), validators
└── types/

web/
├── app/          App Router routes, layout, sitemap.ts, robots.ts, not-found.tsx
├── components/   blocks (one per builder block), layout, ui
├── lib/          api client, seo helpers, formatters
└── public/       flags, static icons
```

Admin stack: React 19, Vite, TypeScript, TanStack Query, TanStack Table, React Hook Form with Zod, TipTap, Tailwind CSS, Recharts. Web stack: Next.js (App Router), TypeScript, Tailwind CSS, `next/font`, `next/image`.

### A.4 Coding rules

- No comments in production code: no explanatory notes, TODO tags, commented-out code or decorative banners. Names and small functions carry the meaning. Exceptions: test files may use comments, and doc comments required by linters on exported Go identifiers are kept to one line.
- Go: `gofmt` and `golangci-lint` clean; errors wrapped with `%w` and context; `context.Context` is the first parameter of any I/O function; no package-level mutable state; no `panic` in request paths.
- TypeScript: `strict` on, no `any`, ESLint clean.
- Keep functions small and single-purpose; prefer early returns.
- Every business rule in PRD Section 11 has a unit test.
- Never reduce, skip or defer data fetching to gain performance; optimize with caching, parallelism and smaller payloads.
- No loading placeholders that hide renderable data; spinners only for slow actions (PRD 14.5).
- Animations follow PRD 14.6: CSS only, 150–300 ms, respect reduced motion.
- Secrets never enter the repository, logs or client bundles.

### A.5 Git and commits

- Commit messages are free-form but must describe what changed and why in plain English, e.g. `Add relief plan conflict check for overlapping dates`.
- Never add co-author trailers or attribution for Anthropic, Claude or any AI tool (no `Co-Authored-By`, no `Generated with` lines).
- One logical change per commit; no unrelated formatting changes mixed in.
- CI must pass before merge.
