import * as core from '@actions/core';
import { createAppAuth } from '@octokit/auth-app';

async function run(): Promise<void> {
  try {
    const appId = core.getInput('app-id', { required: true });
    const privateKey = core.getInput('private-key', { required: true });
    const installationId = core.getInput('installation-id', { required: true });

    const auth = createAppAuth({
      appId,
      privateKey,
      installationId: parseInt(installationId)
    });

    const { token } = await auth({ type: 'installation' });


    core.setSecret(token); // to ensure that the token is masked.
    core.setOutput('token', token);
  } catch (error) {
    if (error instanceof Error) core.setFailed(error.message);
  }
}

run();
