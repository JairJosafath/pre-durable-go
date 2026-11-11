import * as cdk from 'aws-cdk-lib/core';
import { Match, Template } from 'aws-cdk-lib/assertions';
import { PrimeDriveStack } from '../lib/prime-drive-stack';

// Skip compiling the Go binaries: these tests only look at the template.
const app = new cdk.App({ context: { 'aws:cdk:bundling-stacks': [] } });
const template = Template.fromStack(new PrimeDriveStack(app, 'TestStack'));

const durableFunction = { DurableConfig: Match.anyValue() };

describe('upload pipeline', () => {
  test('exactly one function is durable, and it can wait a day for a reviewer', () => {
    template.resourcePropertiesCountIs('AWS::Lambda::Function', durableFunction, 1);
    template.hasResourceProperties('AWS::Lambda::Function', {
      DurableConfig: { ExecutionTimeout: 25 * 60 * 60, RetentionPeriodInDays: 14 },
      Runtime: 'provided.al2023',
      Handler: 'bootstrap',
      Architectures: ['arm64'],
    });
  });

  test('the durable function gets the durable execution policy', () => {
    template.hasResourceProperties('AWS::IAM::Role', {
      ManagedPolicyArns: Match.arrayWith([
        Match.objectLike({
          'Fn::Join': Match.arrayWith([
            Match.arrayWith([':iam::aws:policy/service-role/AWSLambdaBasicDurableExecutionRolePolicy']),
          ]),
        }),
      ]),
    });
  });

  test('S3 starts executions through the live alias, without a trigger function', () => {
    template.hasResourceProperties('AWS::Lambda::Alias', { Name: 'live' });
    template.hasResourceProperties('Custom::S3BucketNotifications', {
      NotificationConfiguration: {
        LambdaFunctionConfigurations: [
          Match.objectLike({
            Events: ['s3:ObjectCreated:*'],
            Filter: { Key: { FilterRules: [{ Name: 'prefix', Value: 'users/' }] } },
            LambdaFunctionArn: { Ref: Match.stringLikeRegexp('UploadPipelineLive') },
          }),
        ],
      },
    });
  });

  test('there is no state machine left', () => {
    template.resourceCountIs('AWS::StepFunctions::StateMachine', 0);
  });

  test('the workflow knows which function to invoke for thumbnails', () => {
    template.hasResourceProperties('AWS::Lambda::Function', {
      ...durableFunction,
      Environment: {
        Variables: Match.objectLike({
          THUMBNAILER_FUNCTION: { Ref: Match.stringLikeRegexp('UploadPipelineThumbnailerCurrentVersion') },
          REVIEW_TIMEOUT: '86400s',
        }),
      },
    });
  });
});

describe('api', () => {
  test('only the reviewer may complete review callbacks', () => {
    const statement = Match.objectLike({
      Action: ['lambda:SendDurableExecutionCallbackSuccess', 'lambda:SendDurableExecutionCallbackFailure'],
    });
    template.resourcePropertiesCountIs(
      'AWS::IAM::Policy',
      { PolicyDocument: { Statement: Match.arrayWith([statement]) } },
      1,
    );
  });

  test('every route except auth and CORS preflight requires a signed-in user', () => {
    const methods = Object.values(template.findResources('AWS::ApiGateway::Method')).map((m) => m.Properties);
    const open = methods.filter((m) => m.AuthorizationType === 'NONE' && m.HttpMethod !== 'OPTIONS');

    expect(open).toHaveLength(3); // login, signup, confirm
    expect(methods.filter((m) => m.AuthorizationType === 'COGNITO_USER_POOLS')).toHaveLength(6);
  });
});
