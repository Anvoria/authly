# Authly Frontend

The official web client for Authly, providing a modern user interface for authentication, registration, and OIDC authorization flows.

## Tech Stack

- **Framework**: Next.js (App Router)
- **Library**: React
- **Styling**: Tailwind CSS
- **State Management**: TanStack Query
- **API Client**: Axios
- **Validation**: Zod
- **Icons**: Lucide React

## Getting Started

### Prerequisites
- [Bun](https://bun.sh/) (preferred) or Node.js 20+

### Development

1. Install dependencies:
   ```bash
   bun install
   ```

2. Configure environment variables:
   Copy `.env.example` to `.env.local` and update the values:
   ```bash
   cp .env.example .env.local
   ```

3. Run the development server:
   ```bash
   bun dev
   ```

The application will be available at http://localhost:3000.

### Production Build

To create an optimized production build:

```bash
bun run build
bun start
```

## Code Quality

Run linting and formatting to ensure code consistency:

```bash
bun run lint
bun run format
```