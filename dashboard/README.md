# MailPulse Dashboard (React)

A modern, professional dashboard for monitoring and managing your MailPulse SMTP relay server.

## Features

- **Modern UI**: Clean, professional interface built with React + Vite + Tailwind CSS
- **Real-time Monitoring**: Live email activity tracking and relay status monitoring
- **Project Management**: Create, configure, and manage multiple SMTP projects
- **Email Analytics**: Detailed email statistics and delivery monitoring
- **Responsive Design**: Works seamlessly on desktop, tablet, and mobile devices
- **Professional Styling**: Modern design with smooth animations and intuitive UX

## Tech Stack

- **Frontend**: React 19 + TypeScript
- **Build Tool**: Vite (fast development and building)
- **Styling**: Tailwind CSS with custom component classes
- **Icons**: Heroicons
- **HTTP Client**: Native Fetch API
- **Development**: Hot Module Replacement (HMR) for fast development

## Getting Started

### Prerequisites

- Node.js 18+ 
- npm or yarn
- MailPulse SMTP relay server running on localhost:8080

### Installation

1. **Install dependencies**
   ```bash
   npm install
   ```

2. **Configure environment**
   ```bash
   cp .env.example .env
   ```
   
   Update `.env` with your configuration:
   ```env
   VITE_RELAY_API_URL=http://localhost:8080
   VITE_APP_ENV=development
   ```

3. **Start development server**
   ```bash
   npm run dev
   ```
   
   The dashboard will be available at http://localhost:3000

### Build for Production

```bash
npm run build
npm run preview  # Preview production build
```

## Project Structure

```
src/
├── components/          # React components
│   ├── Dashboard.tsx   # Main dashboard with overview stats
│   ├── EmailActivity.tsx # Email monitoring and filtering
│   ├── Projects.tsx    # Project management interface
│   ├── EmailDetailModal.tsx # Email content viewer
│   └── CreateProjectModal.tsx # Project creation form
├── lib/
│   └── api.ts          # API client and TypeScript interfaces
├── index.css           # Tailwind CSS and custom styles
├── App.tsx             # Main app component with navigation
└── main.tsx            # React app entry point
```

## Features Overview

### Dashboard
- Real-time email statistics across all projects
- System health monitoring
- Quick SMTP setup guide
- Project performance overview

### Email Activity
- Live email monitoring with real-time updates
- Advanced filtering by project, status, and search
- Email detail viewer with parsed content
- Resend functionality for failed emails
- Email content parsing (handles Base64 encoding)

### Project Management
- Create and configure SMTP projects
- API key generation and management
- Quota monitoring and usage tracking
- Project activation/deactivation
- Real-time status updates

### Professional UI Features
- Collapsible sidebar navigation
- Loading states and smooth animations
- Responsive grid layouts
- Status badges and progress indicators
- Modal overlays with proper focus management
- Custom scrollbars and hover effects

## API Integration

The dashboard connects to the MailPulse Go SMTP relay server via REST API:

- `GET /health` - Server health status
- `GET /api/projects` - List all projects
- `POST /api/projects` - Create new project
- `PATCH /api/projects/{id}` - Update project
- `DELETE /api/projects/{id}` - Delete project
- `GET /api/emails` - List emails with filtering
- `POST /api/emails/{id}/resend` - Resend email
- `GET /api/quota/{projectId}` - Get quota usage
- `GET /api/emails/stats/{projectId}` - Get email statistics

## Development

### Available Scripts

- `npm run dev` - Start development server with HMR
- `npm run build` - Build for production
- `npm run preview` - Preview production build
- `npm run lint` - Run ESLint

### Code Style

- TypeScript for type safety
- Functional components with hooks
- Custom CSS classes for consistent styling
- Responsive-first design approach
- Component composition over inheritance

## Deployment

The built application is a static SPA that can be deployed to any web server:

1. Run `npm run build`
2. Deploy the `dist/` folder contents
3. Configure your web server to serve `index.html` for all routes
4. Set the `VITE_RELAY_API_URL` environment variable for your production API

## Contributing

1. Follow the existing code style and patterns
2. Use TypeScript for all new code
3. Follow the component structure and naming conventions
4. Test your changes with the Go SMTP relay server
5. Ensure responsive design works on all screen sizes

## License

This project is part of the MailPulse SMTP relay system.