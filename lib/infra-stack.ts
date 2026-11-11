import { Duration, Stack, StackProps } from 'aws-cdk-lib/core';
import {Function, Runtime, Code, Architecture} from 'aws-cdk-lib/aws-lambda'
import { Construct } from 'constructs';
import path from 'path';

export class InfraStack extends Stack {
  constructor(scope: Construct, id: string, props?: StackProps) {
    super(scope, id, props);

    const fn = new Function(this, 'PreDurableFunction', {
      runtime: Runtime.PROVIDED_AL2023,
      architecture: Architecture.ARM_64,
      handler: 'bootstrap',
      code: Code.fromAsset(path.join(__dirname, '..', 'src')),
      durableConfig: { executionTimeout: Duration.hours(1), retentionPeriod: Duration.days(30) }, // 1 hour timeout, 30 days retention
    });

  }
}
