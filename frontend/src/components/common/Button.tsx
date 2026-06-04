import { forwardRef, type ButtonHTMLAttributes } from 'react';
import { getButtonClassName, type ButtonSize, type ButtonVariant } from './buttonStyles';

export type { ButtonSize, ButtonVariant } from './buttonStyles';

interface ButtonProps extends ButtonHTMLAttributes<HTMLButtonElement> {
  variant?: ButtonVariant;
  size?: ButtonSize;
}

export const Button = forwardRef<HTMLButtonElement, ButtonProps>(
  ({ variant = 'primary', size = 'md', type = 'button', className, ...props }, ref) => (
    <button
      ref={ref}
      type={type}
      className={getButtonClassName({ variant, size, className })}
      {...props}
    />
  )
);

Button.displayName = 'Button';
