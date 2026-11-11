#!/usr/bin/env node
import * as cdk from 'aws-cdk-lib/core';
import { PrimeDriveStack } from '../lib/prime-drive-stack';

const app = new cdk.App();

// Deploys to the account and region of the current CLI profile. Pick a region
// that offers both Lambda durable functions and Amazon Rekognition.
new PrimeDriveStack(app, 'PrimeDriveStack', {
  env: { account: process.env.CDK_DEFAULT_ACCOUNT, region: process.env.CDK_DEFAULT_REGION },
});
