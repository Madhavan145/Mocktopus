import { cn } from './cn';

interface KbdProps {
  children: React.ReactNode;
  className?: string;
}

export function Kbd({ children, className }: KbdProps) {
  return (
    <kbd
      className={cn(
        'inline-flex h-5 items-center rounded border bg-surface-elevated px-1.5 text-[10px] font-medium text-ink-muted',
        className
      )}
    >
      {children}
    </kbd>
  );
}
