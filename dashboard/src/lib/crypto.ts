/**
 * Generates a secure password using crypto randomness
 * Format: 4 parts separated by hyphens, 3 words (exactly 4 letters) + 1 number (4 digits)
 * Number can be in any position (1st, 2nd, 3rd, or 4th)
 */
export function generatePassword(): string {
  // Generate 3 words (exactly 4 letters each)
  const word1 = generateRandomWord(4)
  const word2 = generateRandomWord(4) 
  const word3 = generateRandomWord(4)
  
  // Generate 4-digit number using crypto random
  const number = 1000 + Math.floor(Math.random() * 9000)
  
  // Randomly choose position for number (0-3)
  const numberPosition = Math.floor(Math.random() * 4)
  
  // Create array of parts and insert number at random position
  const parts = [word1, word2, word3]
  parts.splice(numberPosition, 0, number.toString())
  
  return parts.join('-')
}

/**
 * Generates a random pronounceable word of exactly specified length
 */
function generateRandomWord(exactLength: number): string {
  const consonants = 'bcdfghjklmnpqrstvwxyz'
  const vowels = 'aeiou'
  
  let word = ''
  
  for (let i = 0; i < exactLength; i++) {
    const useVowel = i % 2 === 1 // Alternate consonant-vowel pattern
    const charSet = useVowel ? vowels : consonants
    const randomIndex = Math.floor(Math.random() * charSet.length)
    word += charSet[randomIndex]
  }
  
  
  return word
}