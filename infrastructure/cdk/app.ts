#!/usr/bin/env node
import * as cdk from 'aws-cdk-lib';
import { MskDelayTestStack } from './msk-delay-test-stack';

const app = new cdk.App();
new MskDelayTestStack(app, 'MskDelayTestStack', {
  env: { 
    account: process.env.CDK_DEFAULT_ACCOUNT, 
    region: process.env.CDK_DEFAULT_REGION || 'us-east-1'
  },
});
