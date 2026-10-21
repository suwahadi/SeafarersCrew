import { Controller, type Control, type FieldErrors, type UseFormRegister } from 'react-hook-form'
import { FileField } from '@/components/forms/FileField'
import { SelectField, TextAreaField, TextField } from '@/components/ui/Field'
import { useOptions } from '@/hooks/useOptions'
import type { ContactValues, PersonalValues } from './crewSchema'

const GENDERS = [
  { value: 'male', label: 'Male' },
  { value: 'female', label: 'Female' },
]

const MARITAL = [
  { value: 'single', label: 'Single' },
  { value: 'married', label: 'Married' },
  { value: 'divorced', label: 'Divorced' },
  { value: 'widowed', label: 'Widowed' },
]

interface PersonalProps {
  register: UseFormRegister<PersonalValues>
  control: Control<PersonalValues>
  errors: FieldErrors<PersonalValues>
  age?: number | null
  withFiles?: boolean
}

export function PersonalFields({ register, control, errors, age, withFiles = true }: PersonalProps) {
  const { options } = useOptions()
  const ranks = options.ranks.map((r) => ({ value: r.id, label: `${r.name} (${r.short_code})` }))
  return (
    <div className="grid grid-cols-1 gap-4 md:grid-cols-2 xl:grid-cols-3">
      <TextField label="Full name" required error={errors.full_name?.message} {...register('full_name')} />
      <TextField label="Place of birth" error={errors.place_of_birth?.message} {...register('place_of_birth')} />
      <TextField
        label="Date of birth"
        type="date"
        hint={age !== undefined && age !== null ? `Age ${String(age)}` : undefined}
        error={errors.date_of_birth?.message}
        {...register('date_of_birth')}
      />
      <SelectField label="Gender" placeholder="Not set" options={GENDERS} error={errors.gender?.message} {...register('gender')} />
      <TextField label="Nationality" error={errors.nationality?.message} {...register('nationality')} />
      <TextField label="Religion" error={errors.religion?.message} {...register('religion')} />
      <SelectField label="Marital status" placeholder="Not set" options={MARITAL} error={errors.marital_status?.message} {...register('marital_status')} />
      <TextField label="Height (cm)" inputMode="numeric" error={errors.height_cm?.message} {...register('height_cm')} />
      <TextField label="Weight (kg)" inputMode="numeric" error={errors.weight_kg?.message} {...register('weight_kg')} />
      <TextField label="Safety shoe size" error={errors.shoe_size?.message} {...register('shoe_size')} />
      <TextField label="Coverall size" error={errors.coverall_size?.message} {...register('coverall_size')} />
      <SelectField label="Applied rank" placeholder="Not set" options={ranks} error={errors.applied_rank_id?.message} {...register('applied_rank_id')} />
      {withFiles ? (
        <>
          <Controller
            control={control}
            name="photo_file_id"
            render={({ field, fieldState }) => (
              <FileField
                label="Profile photo"
                context="photo"
                fileId={field.value}
                onChange={(id) => field.onChange(id)}
                hint="JPG, PNG or WebP; stored as WebP without metadata."
                error={fieldState.error?.message}
              />
            )}
          />
          <Controller
            control={control}
            name="cv_file_id"
            render={({ field, fieldState }) => (
              <FileField
                label="External CV"
                context="crew_cv"
                fileId={field.value}
                onChange={(id) => field.onChange(id)}
                hint="PDF, image, DOC or DOCX; Word files are converted to PDF."
                error={fieldState.error?.message}
              />
            )}
          />
        </>
      ) : null}
    </div>
  )
}

interface ContactProps {
  register: UseFormRegister<ContactValues>
  errors: FieldErrors<ContactValues>
}

export function ContactFields({ register, errors }: ContactProps) {
  return (
    <div className="grid grid-cols-1 gap-4 md:grid-cols-2 xl:grid-cols-3">
      <div className="md:col-span-2 xl:col-span-3">
        <TextAreaField label="Full address" rows={2} error={errors.address?.message} {...register('address')} />
      </div>
      <TextField label="City" error={errors.city?.message} {...register('city')} />
      <TextField label="Province" error={errors.province?.message} {...register('province')} />
      <TextField label="Country" error={errors.country?.message} {...register('country')} />
      <TextField label="Postal code" error={errors.postal_code?.message} {...register('postal_code')} />
      <TextField label="Mobile phone" type="tel" placeholder="+628123456789" error={errors.mobile_phone?.message} {...register('mobile_phone')} />
      <TextField label="Home phone" type="tel" placeholder="+62215550100" error={errors.home_phone?.message} {...register('home_phone')} />
      <TextField label="Email" type="email" error={errors.email?.message} {...register('email')} />
      <TextField label="Nearest airport" error={errors.nearest_airport?.message} {...register('nearest_airport')} />
    </div>
  )
}
