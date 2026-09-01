/**
 * Advanced smart punctuation, segmentation, and capitalization engine
 * for voice-to-text transcription.
 */

const QUESTION_WORDS = [
  'what',
  'why',
  'how',
  'when',
  'where',
  'who',
  'which',
  'whose',
  'is',
  'are',
  'can',
  'could',
  'should',
  'would',
  'does',
  'do',
  'did',
  'will',
  'shall',
  'may',
  'might',
  'have',
  'has',
  'had',
  'tell me',
  'explain',
  'describe',
]

const ACRONYMS: Record<string, string> = {
  fcr: 'FCR',
  ph: 'pH',
  do: 'DO',
  tan: 'TAN',
  nh3: 'NH3',
  nh4: 'NH4',
  no2: 'NO2',
  no3: 'NO3',
  fao: 'FAO',
  ras: 'RAS',
  ppm: 'ppm',
  ppt: 'ppt',
  ntu: 'NTU',
  'african catfish': 'African catfish',
  'nile tilapia': 'Nile tilapia',
  'clarias gariepinus': 'Clarias gariepinus',
  'oreochromis niloticus': 'Oreochromis niloticus',
}

/**
 * Replace spoken punctuation commands if the speaker verbally dictates punctuation.
 */
function replaceSpokenPunctuation(text: string): string {
  return text
    .replace(/\s+comma\b/gi, ',')
    .replace(/\s+(full stop|period)\b/gi, '.')
    .replace(/\s+question mark\b/gi, '?')
    .replace(/\s+(exclamation mark|exclamation point)\b/gi, '!')
    .replace(/\s+colon\b/gi, ':')
    .replace(/\s+semicolon\b/gi, ';')
    .replace(/\s+(new line|next line)\b/gi, '\n')
}

/**
 * Formats numbers, units, and temperatures.
 */
function formatUnitsAndNumbers(text: string): string {
  return text
    .replace(/\b(\d+)\s+degrees?\s*(celsius|c)?\b/gi, '$1°C')
    .replace(/\bpoint\s+(\d+)\b/gi, '0.$1')
    .replace(/\b(\d+)\s+point\s+(\d+)\b/gi, '$1.$2')
    .replace(/\bpond\s+([a-zA-Z0-9]+)\b/gi, (_m, p) => `Pond ${p.toUpperCase()}`)
}

/**
 * Inserts commas for transitions and between independent clauses.
 */
function insertNaturalCommas(text: string): string {
  const transitions = [
    'for example',
    'in addition',
    'however',
    'therefore',
    'moreover',
    'on the other hand',
    'as a result',
    'in fact',
    'meanwhile',
    'furthermore',
    'at the same time',
  ]

  for (const phrase of transitions) {
    const regex = new RegExp(`\\b(${phrase})\\b`, 'gi')
    text = text.replace(regex, ', $1,')
  }

  // Comma before coordinating conjunctions when bridging clauses
  const conjunctions = ['and also', 'but', 'so that', 'although', 'because', 'whereas', 'while', 'however']
  for (const conj of conjunctions) {
    const regex = new RegExp(`([a-zA-Z0-9]{3,})\\s+(${conj})\\s+([a-zA-Z])`, 'gi')
    text = text.replace(regex, '$1, $2 $3')
  }

  // Clean up duplicate commas or commas before sentence terminators
  text = text.replace(/\s*,\s*/g, ', ')
  text = text.replace(/,\s*,+/g, ', ')
  text = text.replace(/^,\s*/g, '')
  text = text.replace(/,\s*([.?!])/g, '$1')

  return text
}

/**
 * Splits compound spoken streams into distinct punctuated sentences.
 * e.g. "my fish are not eating what should I do" -> "My fish are not eating. What should I do?"
 */
function segmentCompoundQuestions(text: string): string {
  for (const qWord of ['what', 'why', 'how', 'when', 'where', 'is there', 'can you', 'should I', 'what should', 'how can', 'how much', 'how many']) {
    const regex = new RegExp(`([a-zA-Z0-9]{3,})\\s+(${qWord}\\b)`, 'gi')
    text = text.replace(regex, (_match, before, questionStart) => {
      // If preceding word isn't a connector like "know", "see", "ask", insert period
      const lowerBefore = before.toLowerCase()
      if (['know', 'see', 'understand', 'determine', 'check', 'if', 'whether'].includes(lowerBefore)) {
        return `${before} ${questionStart}`
      }
      return `${before}. ${questionStart.charAt(0).toUpperCase() + questionStart.slice(1)}`
    })
  }
  return text
}

/**
 * Capitalizes sentences and aquaculture terms, then ensures closing punctuation.
 */
export function formatSpokenText(raw: string): string {
  if (!raw || !raw.trim()) return ''

  let text = raw.trim().replace(/\s+/g, ' ')

  // 1. Spoken punctuation commands
  text = replaceSpokenPunctuation(text)

  // 2. Units and numbers
  text = formatUnitsAndNumbers(text)

  // 3. Segment compound sentences
  text = segmentCompoundQuestions(text)

  // 4. Natural comma insertion
  text = insertNaturalCommas(text)

  // 5. Replace aquaculture domain terms
  for (const [lower, formatted] of Object.entries(ACRONYMS)) {
    const regex = new RegExp(`\\b${lower}\\b`, 'gi')
    text = text.replace(regex, formatted)
  }

  // 6. Sentence capitalization
  text = text.replace(/(^\s*|[.?!]\s+)([a-z])/g, (_match, prefix, char) => {
    return prefix + char.toUpperCase()
  })

  // 7. Final punctuation
  const lastChar = text.slice(-1)
  if (!['.', '?', '!', ';', ':'].includes(lastChar)) {
    const sentences = text.split(/[.?!]\s+/)
    const lastSentence = (sentences[sentences.length - 1] || '').trim().toLowerCase()

    const isQuestion = QUESTION_WORDS.some(
      (starter) => lastSentence.startsWith(starter + ' ') || lastSentence === starter,
    )
    text += isQuestion ? '?' : '.'
  }

  return text
}
