import { z } from 'zod'
import type { Crew, CrewInput } from '@/api/crews'

const e164 = /^\+[1-9][0-9]{6,14}$/
const isoDate = /^\d{4}-\d{2}-\d{2}$/

const optionalPhone = z.string().trim().refine((v) => v === '' || e164.test(v), 'Use E.164 format, e.g. +628123456789.')
const optionalDate = z.string().refine((v) => v === '' || isoDate.test(v), 'Use a valid date.')
const optionalRange = (min: number, max: number) =>
  z
    .string()
    .trim()
    .refine((v) => v === '' || (/^\d+$/.test(v) && Number(v) >= min && Number(v) <= max), `Enter a number between ${String(min)} and ${String(max)}.`)

export const personalSchema = z.object({
  full_name: z.string().trim().min(1, 'The full name is required.').max(150, 'At most 150 characters.'),
  place_of_birth: z.string().trim().max(100),
  date_of_birth: optionalDate,
  gender: z.enum(['', 'male', 'female']),
  nationality: z.string().trim().max(100),
  religion: z.string().trim().max(100),
  marital_status: z.enum(['', 'single', 'married', 'divorced', 'widowed']),
  height_cm: optionalRange(50, 250),
  weight_kg: optionalRange(20, 300),
  shoe_size: z.string().trim().max(10),
  coverall_size: z.string().trim().max(10),
  applied_rank_id: z.string(),
  photo_file_id: z.string(),
  cv_file_id: z.string(),
})

export const contactSchema = z.object({
  address: z.string().trim().max(500),
  city: z.string().trim().max(100),
  province: z.string().trim().max(100),
  country: z.string().trim().max(100),
  postal_code: z.string().trim().max(20),
  home_phone: optionalPhone,
  mobile_phone: optionalPhone,
  email: z
    .string()
    .trim()
    .refine((v) => v === '' || z.email().safeParse(v).success, 'Enter a valid email address.'),
  nearest_airport: z.string().trim().max(100),
})

export type PersonalValues = z.infer<typeof personalSchema>
export type ContactValues = z.infer<typeof contactSchema>

export const PERSONAL_FIELDS = Object.keys(personalSchema.shape) as (keyof PersonalValues)[]
export const CONTACT_FIELDS = Object.keys(contactSchema.shape) as (keyof ContactValues)[]

export const EMPTY_PERSONAL: PersonalValues = {
  full_name: '',
  place_of_birth: '',
  date_of_birth: '',
  gender: '',
  nationality: '',
  religion: '',
  marital_status: '',
  height_cm: '',
  weight_kg: '',
  shoe_size: '',
  coverall_size: '',
  applied_rank_id: '',
  photo_file_id: '',
  cv_file_id: '',
}

export const EMPTY_CONTACT: ContactValues = {
  address: '',
  city: '',
  province: '',
  country: '',
  postal_code: '',
  home_phone: '',
  mobile_phone: '',
  email: '',
  nearest_airport: '',
}

export function personalFromCrew(c: Crew): PersonalValues {
  return {
    full_name: c.full_name,
    place_of_birth: c.place_of_birth,
    date_of_birth: c.date_of_birth ?? '',
    gender: c.gender ?? '',
    nationality: c.nationality,
    religion: c.religion,
    marital_status: c.marital_status ?? '',
    height_cm: c.height_cm === null ? '' : String(c.height_cm),
    weight_kg: c.weight_kg === null ? '' : String(c.weight_kg),
    shoe_size: c.shoe_size,
    coverall_size: c.coverall_size,
    applied_rank_id: c.applied_rank_id ?? '',
    photo_file_id: c.photo_file_id ?? '',
    cv_file_id: c.cv_file_id ?? '',
  }
}

export function contactFromCrew(c: Crew): ContactValues {
  return {
    address: c.address,
    city: c.city,
    province: c.province,
    country: c.country,
    postal_code: c.postal_code,
    home_phone: c.home_phone,
    mobile_phone: c.mobile_phone,
    email: c.email,
    nearest_airport: c.nearest_airport,
  }
}

function num(v: string): number | null {
  return v.trim() === '' ? null : Number(v)
}

export function toCrewInput(p: PersonalValues, c: ContactValues): CrewInput {
  return { ...p, ...c, height_cm: num(p.height_cm), weight_kg: num(p.weight_kg) }
}
