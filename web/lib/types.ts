export interface ImageSource {
  url: string
  width: number
  height: number
}

export interface PublicImage {
  url: string
  mime_type: string
  width: number
  height: number
  alt: string
  sources: ImageSource[]
}

export type MaybeImage = PublicImage | null

export interface MenuItem {
  label: string
  href: string
  open_in_new_tab: boolean
  children: MenuItem[] | null
}

export interface Language {
  name: string
  code: string
  flag_code: string
  flag: MaybeImage
  is_source: boolean
}

export interface Site {
  name: string
  tagline: string
  default_language: string
  maintenance_mode: boolean
  organization: {
    legal_name: string
    website: string
    email: string
    phone: string
    address: string
    country: string
  }
  branding: {
    logo_light: MaybeImage
    logo_dark: MaybeImage
    favicon: MaybeImage
    primary_color: string
    accent_color: string
    footer_text: string
  }
  contact: {
    office_address: string
    coordinates: { lat: number; lng: number } | null
    phones: string[] | null
    email: string
    whatsapp: { number: string; message: string; url: string } | null
  }
  social: {
    linkedin: string
    facebook: string
    instagram: string
    youtube: string
    x: string
  }
  seo: {
    meta_title_pattern: string
    meta_description: string
    default_og_image: MaybeImage
    google_site_verification: string
    robots_indexing: boolean
  }
  menus: { header: MenuItem[] | null; footer: MenuItem[] | null }
  languages: Language[] | null
  hero_autoplay_seconds: number
  analytics_enabled: boolean
  turnstile_site_key: string
}

export interface NewsCard {
  title: string
  slug: string
  excerpt: string
  published_at: string
  thumbnail: MaybeImage
}

export interface NewsArticle extends NewsCard {
  body_html: string
  seo_title: string
  meta_description: string
  updated_at: string
  related: NewsCard[] | null
}

export interface Rank {
  id: string
  name: string
}

export interface VacancyCard {
  id: string
  title: string
  slug: string
  rank: Rank
  vessel_type: Rank | null
  department: string
  location: string
  employment_type: 'contract' | 'permanent'
  closing_date: string | null
  published_at: string
}

export interface GalleryItem {
  id: string
  caption: string
  image: MaybeImage
}

export interface HeroSlide {
  id: string
  heading: string
  subheading: string
  button_label: string
  button_link: string
  image: MaybeImage
  mobile_image: MaybeImage
}

export interface StatItem {
  number: number
  suffix: string
  label: string
}

export interface Testimonial {
  id: string
  name: string
  position: string
  photo: MaybeImage
  quote: string
  rating: number
}

export interface ClientLogo {
  id: string
  name: string
  url: string
  logo: MaybeImage
}

export interface PageHead {
  title: string
  seo_title: string
  meta_description: string
  og_image: MaybeImage
  intro_html: string
}

export type HomeSectionKey =
  | 'hero'
  | 'intro'
  | 'stats'
  | 'services'
  | 'news'
  | 'vacancies'
  | 'gallery'
  | 'testimonials'
  | 'clients'
  | 'cta'

export interface Home {
  page: PageHead | null
  sections: string[] | null
  hero: { autoplay_seconds: number; slides: HeroSlide[] | null }
  stats: (StatItem & { id: string })[] | null
  news: NewsCard[] | null
  vacancies: VacancyCard[] | null
  gallery: GalleryItem[] | null
  testimonials: Testimonial[] | null
  client_logos: ClientLogo[] | null
}

export interface RichTextSettings {
  html: string
}

export interface ImageTextSettings {
  image: MaybeImage
  heading: string
  text_html: string
  button_label: string
  button_link: string
  image_position: 'left' | 'right'
}

export interface GallerySettings {
  images: GalleryItem[] | null
  columns: number
  lightbox: boolean
}

export interface VideoSettings {
  provider: 'youtube' | 'vimeo'
  video_id: string
  title: string
  url: string
  embed_url: string
  thumbnail_url: string
}

export interface CtaSettings {
  heading: string
  text: string
  button_label: string
  button_link: string
  background_image: MaybeImage
  background_color: string
}

export interface StatsSettings {
  items: StatItem[] | null
}

export interface FaqSettings {
  items: { question: string; answer_html: string }[] | null
}

export interface Card {
  icon: string
  image: MaybeImage
  title: string
  text: string
  link: string
}

export interface CardGridSettings {
  columns: number
  cards: Card[] | null
}

export interface BlockSettingsMap {
  rich_text: RichTextSettings
  image_text: ImageTextSettings
  gallery: GallerySettings
  video: VideoSettings
  cta: CtaSettings
  stats: StatsSettings
  faq: FaqSettings
  card_grid: CardGridSettings
}

export type BlockType = keyof BlockSettingsMap

export type KnownBlock = {
  [K in BlockType]: { id: string; type: K; settings: BlockSettingsMap[K] }
}[BlockType]

export interface RawBlock {
  id: string
  type: string
  settings: unknown
}

export interface CmsPage extends PageHead {
  slug: string
  path: string
  system_key: string | null
  blocks: RawBlock[] | null
  published_at: string | null
  updated_at: string
}

export interface RedirectRule {
  from_path: string
  to_path: string
  status_code: number
}

export interface SitemapEntry {
  path: string
  lastmod: string
}

export interface SitemapData {
  pages: SitemapEntry[] | null
  system_pages: SitemapEntry[] | null
  news: SitemapEntry[] | null
  vacancies: SitemapEntry[] | null
}

export interface ListMeta {
  page: number
  per_page: number
  total: number
}

export interface Paged<T> {
  data: T[]
  meta: ListMeta
}
