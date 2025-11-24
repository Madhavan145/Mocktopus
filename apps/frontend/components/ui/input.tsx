import { forwardRef, type InputHTMLAttributes } from 'react';
import { cn } from './cn';

export interface InputProps extends InputHTMLAttributes<HTMLInputElement> {
  label?: string;
  error?: string;
}

const Input = forwardRef<HTMLInputElement, InputProps>(({ className, label, error, id, ...props }, ref) => {
  return (
    <div className="space-y-1.5">
      {label && (
        <label htmlFor={id} className="text-caption font-medium text-ink-secondary">
          {label}
        </label>
      )}
      <input
        id={id}
        ref={ref}
        className={cn(
          'flex h-9 w-full rounded-md border bg-surface px-3 py-2 text-body-sm text-ink placeholder:text-ink-muted transition-colors duration-150',
          'focus:border-accent focus:ring-1 focus:ring-accent focus:outline-none',
          'hover:border-ink-muted',
          error && 'border-error focus:border-error focus:ring-error',
          className
        )}
        {...props}
      />
      {error && <p className="text-caption text-error">{error}</p>}
    </div>
  );
});
Input.displayName = 'Input';

export { Input };
