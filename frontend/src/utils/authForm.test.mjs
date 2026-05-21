import assert from 'node:assert/strict';
import { rm } from 'node:fs/promises';
import { tmpdir } from 'node:os';
import path from 'node:path';
import test from 'node:test';
import { build } from 'esbuild';

const tempDir = path.join(tmpdir(), 'wendao-auth-form-tests');
const bundlePath = path.join(tempDir, 'authForm.test-bundle.mjs');

const loadAuthForm = async () => {
  await build({
    entryPoints: [new URL('./authForm.ts', import.meta.url).pathname],
    bundle: true,
    format: 'esm',
    outfile: bundlePath,
    platform: 'node',
  });

  return import(`file://${bundlePath}?cache=${Date.now()}`);
};

test.after(async () => {
  await rm(tempDir, { recursive: true, force: true });
});

test('validateLoginForm requires visible email and password values', async () => {
  const { validateLoginForm } = await loadAuthForm();

  const result = validateLoginForm({ email: '   ', password: '' });

  assert.equal(result.isValid, false);
  assert.equal(result.fieldErrors.email, 'auth.emailRequired');
  assert.equal(result.fieldErrors.password, 'auth.passwordRequired');
});

test('validateLoginForm rejects malformed email before submit', async () => {
  const { validateLoginForm } = await loadAuthForm();

  const result = validateLoginForm({ email: 'not-an-email', password: 'secret123' });

  assert.equal(result.isValid, false);
  assert.equal(result.fieldErrors.email, 'auth.emailInvalid');
});

test('validateRegisterForm trims accepted values', async () => {
  const { validateRegisterForm } = await loadAuthForm();

  const result = validateRegisterForm({
    username: '  lizi  ',
    email: '  lizi@example.com  ',
    password: 'secret123',
    confirmPassword: 'secret123',
    verificationCode: ' 123456 ',
  });

  assert.equal(result.isValid, true);
  assert.equal(result.values.username, 'lizi');
  assert.equal(result.values.email, 'lizi@example.com');
  assert.equal(result.values.verificationCode, '123456');
  assert.deepEqual(result.fieldErrors, {});
});

test('validateRegisterForm reports username password and confirmation issues', async () => {
  const { validateRegisterForm } = await loadAuthForm();

  const result = validateRegisterForm({
    username: 'a',
    email: 'lizi@example.com',
    password: '123',
    confirmPassword: '1234',
    verificationCode: 'abc',
  });

  assert.equal(result.isValid, false);
  assert.equal(result.fieldErrors.username, 'auth.usernameTooShort');
  assert.equal(result.fieldErrors.password, 'auth.passwordTooShort');
  assert.equal(result.fieldErrors.confirmPassword, 'auth.passwordMismatch');
  assert.equal(result.fieldErrors.verificationCode, 'auth.verificationCodeInvalid');
});

test('validatePasswordResetForm requires matching password and verification code', async () => {
  const { validatePasswordResetForm } = await loadAuthForm();

  const result = validatePasswordResetForm({
    email: 'reset@example.com',
    password: 'secret123',
    confirmPassword: 'secret123',
    verificationCode: '654321',
  });

  assert.equal(result.isValid, true);
  assert.deepEqual(result.fieldErrors, {});
});

test('mapAuthErrorToForm turns known backend auth errors into form feedback', async () => {
  const { mapAuthErrorToForm } = await loadAuthForm();

  assert.deepEqual(mapAuthErrorToForm('Invalid email or password', 'login'), {
    target: 'form',
    messageKey: 'auth.invalidCredentials',
  });
  assert.deepEqual(mapAuthErrorToForm('Email already exists', 'register'), {
    target: 'email',
    messageKey: 'auth.emailAlreadyExists',
  });
  assert.deepEqual(mapAuthErrorToForm('Invalid verification code', 'register'), {
    target: 'verificationCode',
    messageKey: 'auth.verificationCodeInvalid',
  });
});
