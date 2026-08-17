/**
 * Minimal, safe renderer for model-authored text.
 *
 * 15_AQUADOC_FRONTEND.md section 10: sanitize rendered Markdown, disallow
 * unrestricted raw HTML, protect against XSS. "Do not render LLM content using
 * unrestricted `dangerouslySetInnerHTML`."
 *
 * The approach here removes the XSS question rather than defending against it:
 * the text is parsed into React elements and every character ends up inside a
 * text node. No HTML string is ever constructed, so there is nothing for a
 * sanitizer to miss. A `<script>` in an answer renders as the literal
 * characters `<script>`.
 *
 * Supported: paragraphs, bullet and numbered lists, `**bold**`, `` `code` ``.
 * Anything else renders as plain text, which is the correct failure mode.
 */

import type { ReactNode } from 'react'

const BOLD_OR_CODE = /(\*\*[^*]+\*\*|`[^`]+`)/g
const BULLET = /^\s*[-*•]\s+(.*)$/
const NUMBERED = /^\s*(\d+)[.)]\s+(.*)$/

/** Render inline `**bold**` and `` `code` `` spans. */
function renderInline(text: string, keyPrefix: string): ReactNode[] {
  return text.split(BOLD_OR_CODE).filter(Boolean).map((part, index) => {
    const key = `${keyPrefix}-${index}`
    if (part.startsWith('**') && part.endsWith('**') && part.length > 4) {
      return <strong key={key}>{part.slice(2, -2)}</strong>
    }
    if (part.startsWith('`') && part.endsWith('`') && part.length > 2) {
      return <code key={key}>{part.slice(1, -1)}</code>
    }
    // Plain text — React escapes it on insertion.
    return <span key={key}>{part}</span>
  })
}

/**
 * Render an answer as React elements.
 *
 * Blocks are separated by blank lines; consecutive list items group into one
 * list so the output reads as intended rather than as one item per paragraph.
 */
export function renderAnswer(text: string): ReactNode {
  const lines = text.replace(/\r\n/g, '\n').split('\n')
  const blocks: ReactNode[] = []

  let paragraph: string[] = []
  let listItems: string[] = []
  let listOrdered = false

  const flushParagraph = () => {
    if (paragraph.length === 0) return
    const key = `p-${blocks.length}`
    blocks.push(<p key={key}>{renderInline(paragraph.join(' '), key)}</p>)
    paragraph = []
  }

  const flushList = () => {
    if (listItems.length === 0) return
    const key = `l-${blocks.length}`
    const items = listItems.map((item, index) => (
      <li key={`${key}-${index}`}>{renderInline(item, `${key}-${index}`)}</li>
    ))
    blocks.push(listOrdered ? <ol key={key}>{items}</ol> : <ul key={key}>{items}</ul>)
    listItems = []
  }

  for (const line of lines) {
    if (line.trim() === '') {
      flushParagraph()
      flushList()
      continue
    }

    const bullet = BULLET.exec(line)
    if (bullet) {
      flushParagraph()
      if (listOrdered) flushList()
      listOrdered = false
      listItems.push(bullet[1] ?? '')
      continue
    }

    const numbered = NUMBERED.exec(line)
    if (numbered) {
      flushParagraph()
      if (!listOrdered) flushList()
      listOrdered = true
      listItems.push(numbered[2] ?? '')
      continue
    }

    flushList()
    paragraph.push(line.trim())
  }

  flushParagraph()
  flushList()

  return <>{blocks}</>
}
