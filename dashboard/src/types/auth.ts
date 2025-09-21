export interface AdminLoginRequest {
  username: string
  password: string
}

export interface AdminLoginResponse {
  token: string
  expiresAt: number
}

export interface AdminVerifyResponse {
  valid: boolean
  username: string
  expiresAt: number
}