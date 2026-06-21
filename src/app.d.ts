import type { AdminUser } from '$lib/types';

declare global {
  namespace App {
    interface Locals {
      user: AdminUser | null;
      sessionId: string | null;
    }
  }
}

export {};
