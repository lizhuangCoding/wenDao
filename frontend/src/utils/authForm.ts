export type AuthField =
  | 'email'
  | 'password'
  | 'username'
  | 'confirmPassword'
  | 'verificationCode';
export type AuthErrorTarget = AuthField | 'form';
export type AuthMode = 'login' | 'register' | 'passwordReset';

export interface LoginFormValues {
  email: string;
  password: string;
}

export interface RegisterFormValues extends LoginFormValues {
  username: string;
  confirmPassword: string;
  verificationCode: string;
}

export interface PasswordResetFormValues extends LoginFormValues {
  confirmPassword: string;
  verificationCode: string;
}

export interface AuthValidationResult<TValues> {
  values: TValues;
  fieldErrors: Partial<Record<AuthField, string>>;
  isValid: boolean;
}

export interface AuthFormFeedback {
  target: AuthErrorTarget;
  messageKey: string;
  fallbackMessage?: string;
}

const emailPattern = /^[^\s@]+@[^\s@]+\.[^\s@]+$/;

const hasErrors = (fieldErrors: Partial<Record<AuthField, string>>) =>
  Object.keys(fieldErrors).length > 0;

export const validateAuthEmail = (email: string) => {
  const normalizedEmail = email.trim();
  const fieldErrors: Partial<Record<AuthField, string>> = {};

  if (!normalizedEmail) {
    fieldErrors.email = 'auth.emailRequired';
  } else if (!emailPattern.test(normalizedEmail)) {
    fieldErrors.email = 'auth.emailInvalid';
  }

  return {
    value: normalizedEmail,
    fieldErrors,
    isValid: !hasErrors(fieldErrors),
  };
};

export const validateLoginForm = (
  values: LoginFormValues
): AuthValidationResult<LoginFormValues> => {
  const normalizedValues = {
    email: values.email.trim(),
    password: values.password,
  };
  const fieldErrors: Partial<Record<AuthField, string>> = {};

  if (!normalizedValues.email) {
    fieldErrors.email = 'auth.emailRequired';
  } else if (!emailPattern.test(normalizedValues.email)) {
    fieldErrors.email = 'auth.emailInvalid';
  }

  if (!normalizedValues.password) {
    fieldErrors.password = 'auth.passwordRequired';
  }

  return {
    values: normalizedValues,
    fieldErrors,
    isValid: !hasErrors(fieldErrors),
  };
};

export const validateRegisterForm = (
  values: RegisterFormValues
): AuthValidationResult<RegisterFormValues> => {
  const normalizedValues = {
    username: values.username.trim(),
    email: values.email.trim(),
    password: values.password,
    confirmPassword: values.confirmPassword,
    verificationCode: values.verificationCode.trim(),
  };
  const fieldErrors: Partial<Record<AuthField, string>> = {};

  if (!normalizedValues.username) {
    fieldErrors.username = 'auth.usernameRequired';
  } else if (normalizedValues.username.length < 2) {
    fieldErrors.username = 'auth.usernameTooShort';
  } else if (normalizedValues.username.length > 50) {
    fieldErrors.username = 'auth.usernameTooLong';
  }

  if (!normalizedValues.email) {
    fieldErrors.email = 'auth.emailRequired';
  } else if (!emailPattern.test(normalizedValues.email)) {
    fieldErrors.email = 'auth.emailInvalid';
  }

  if (!normalizedValues.password) {
    fieldErrors.password = 'auth.passwordRequired';
  } else if (normalizedValues.password.length < 6) {
    fieldErrors.password = 'auth.passwordTooShort';
  }

  if (!normalizedValues.confirmPassword) {
    fieldErrors.confirmPassword = 'auth.confirmPasswordRequired';
  } else if (normalizedValues.password !== normalizedValues.confirmPassword) {
    fieldErrors.confirmPassword = 'auth.passwordMismatch';
  }

  if (!normalizedValues.verificationCode) {
    fieldErrors.verificationCode = 'auth.verificationCodeRequired';
  } else if (!/^\d{6}$/.test(normalizedValues.verificationCode)) {
    fieldErrors.verificationCode = 'auth.verificationCodeInvalid';
  }

  return {
    values: normalizedValues,
    fieldErrors,
    isValid: !hasErrors(fieldErrors),
  };
};

export const validatePasswordResetForm = (
  values: PasswordResetFormValues
): AuthValidationResult<PasswordResetFormValues> => {
  const normalizedValues = {
    email: values.email.trim(),
    password: values.password,
    confirmPassword: values.confirmPassword,
    verificationCode: values.verificationCode.trim(),
  };
  const fieldErrors: Partial<Record<AuthField, string>> = {};

  const emailValidation = validateAuthEmail(normalizedValues.email);
  if (emailValidation.fieldErrors.email) {
    fieldErrors.email = emailValidation.fieldErrors.email;
  }

  if (!normalizedValues.password) {
    fieldErrors.password = 'auth.passwordRequired';
  } else if (normalizedValues.password.length < 6) {
    fieldErrors.password = 'auth.passwordTooShort';
  }

  if (!normalizedValues.confirmPassword) {
    fieldErrors.confirmPassword = 'auth.confirmPasswordRequired';
  } else if (normalizedValues.password !== normalizedValues.confirmPassword) {
    fieldErrors.confirmPassword = 'auth.passwordMismatch';
  }

  if (!normalizedValues.verificationCode) {
    fieldErrors.verificationCode = 'auth.verificationCodeRequired';
  } else if (!/^\d{6}$/.test(normalizedValues.verificationCode)) {
    fieldErrors.verificationCode = 'auth.verificationCodeInvalid';
  }

  return {
    values: normalizedValues,
    fieldErrors,
    isValid: !hasErrors(fieldErrors),
  };
};

export const mapAuthErrorToForm = (
  message: string | undefined,
  mode: AuthMode
): AuthFormFeedback => {
  const normalizedMessage = (message || '').trim();
  const lowerMessage = normalizedMessage.toLowerCase();

  if (lowerMessage.includes('invalid email or password')) {
    return {
      target: 'form',
      messageKey: 'auth.invalidCredentials',
    };
  }

  if (lowerMessage.includes('email already exists')) {
    return {
      target: 'email',
      messageKey: 'auth.emailAlreadyExists',
    };
  }

  if (lowerMessage.includes('verification code')) {
    return {
      target: 'verificationCode',
      messageKey: lowerMessage.includes('recently') ? 'auth.codeTooFrequent' : 'auth.verificationCodeInvalid',
    };
  }

  if (lowerMessage.includes('account is banned')) {
    return {
      target: 'form',
      messageKey: 'auth.accountBanned',
    };
  }

  if (lowerMessage.includes("email' failed") || lowerMessage.includes('required,email')) {
    return {
      target: 'email',
      messageKey: 'auth.emailInvalid',
    };
  }

  if (lowerMessage.includes("password' failed") && lowerMessage.includes('min')) {
    return {
      target: 'password',
      messageKey: 'auth.passwordTooShort',
    };
  }

  if (lowerMessage.includes("username' failed") && lowerMessage.includes('min')) {
    return {
      target: 'username',
      messageKey: 'auth.usernameTooShort',
    };
  }

  return {
    target: 'form',
    messageKey:
      mode === 'login'
        ? 'auth.loginFailed'
        : mode === 'passwordReset'
          ? 'auth.passwordResetFailed'
          : 'auth.registerFailed',
    fallbackMessage: normalizedMessage || undefined,
  };
};
