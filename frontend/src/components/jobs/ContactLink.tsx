export interface ContactLinkProps {
  contact: string
}

type ContactType = 'email' | 'telegram' | 'url' | 'phone' | 'unknown'

const detectContactType = (contact: string): ContactType => {
  const trimmed = contact.trim()

  if (trimmed.startsWith('http://') || trimmed.startsWith('https://')) {
    return 'url'
  }
  if (trimmed.startsWith('@')) {
    return 'telegram'
  }
  if (/^\+?[\d\s-]{7,}$/.test(trimmed)) {
    return 'phone'
  }
  if (trimmed.includes('@') && !trimmed.startsWith('@') && trimmed.includes('.')) {
    return 'email'
  }
  return 'unknown'
}

const getHref = (contact: string, type: ContactType): string | null => {
  const trimmed = contact.trim()

  switch (type) {
    case 'email':
      return `mailto:${trimmed}`
    case 'telegram':
      return `https://t.me/${trimmed.slice(1)}`
    case 'url':
      return trimmed
    case 'phone':
      return `tel:${trimmed.replace(/[\s-]/g, '')}`
    default:
      return null
  }
}

export const ContactLink = ({ contact }: ContactLinkProps) => {
  const type = detectContactType(contact)
  const href = getHref(contact, type)

  if (!href) {
    return <span>{contact}</span>
  }

  const isExternal = type === 'url'

  return (
    <a
      href={href}
      target={isExternal ? '_blank' : undefined}
      rel={isExternal ? 'noopener noreferrer' : undefined}
    >
      {contact}
    </a>
  )
}
