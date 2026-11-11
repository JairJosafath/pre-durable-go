import { Duration } from 'aws-cdk-lib/core';
import { TableV2 } from 'aws-cdk-lib/aws-dynamodb';
import { PolicyStatement } from 'aws-cdk-lib/aws-iam';
import { Alias, IFunction } from 'aws-cdk-lib/aws-lambda';
import { Bucket, EventType } from 'aws-cdk-lib/aws-s3';
import { LambdaDestination } from 'aws-cdk-lib/aws-s3-notifications';
import { Construct } from 'constructs';
import { goLambda } from './go-function';
import { USER_PREFIX } from './storage';

export interface UploadPipelineProps {
  readonly uploads: Bucket;
  readonly thumbnails: Bucket;
  readonly table: TableV2;
  /** How long an upload may wait for a human reviewer before it is rejected. */
  readonly reviewTimeout?: Duration;
}

/**
 * The upload pipeline: one durable Lambda function that replaces the Step
 * Functions state machine, its trigger function, and the ASL definition of the
 * original PrimeDrive project.
 */
export class UploadPipeline extends Construct {
  /** Alias that S3 invokes and that callbacks are sent to. */
  readonly workflow: Alias;
  readonly thumbnailer: IFunction;

  constructor(scope: Construct, id: string, props: UploadPipelineProps) {
    super(scope, id);

    const reviewTimeout = props.reviewTimeout ?? Duration.hours(24);

    // A plain Lambda function. The workflow calls it with durable.Invoke, so
    // the call is checkpointed like any other durable operation.
    const thumbnailer = goLambda(this, 'Thumbnailer', 'thumbnailer', {
      description: 'Resizes an uploaded image into a thumbnail',
      memorySize: 1024,
      timeout: Duration.seconds(30),
      environment: { THUMBNAIL_BUCKET: props.thumbnails.bucketName },
    });
    props.uploads.grantRead(thumbnailer);
    props.thumbnails.grantPut(thumbnailer);

    // durableConfig is the only infrastructure difference between a durable
    // function and a regular one. CDK swaps the basic execution policy for
    // AWSLambdaBasicDurableExecutionRolePolicy, which lets the function
    // checkpoint and read back its own execution state.
    const workflow = goLambda(this, 'Workflow', 'pipeline', {
      description: 'Durable workflow that analyses, reviews and indexes an upload',
      memorySize: 256,
      // One invocation is short; the execution as a whole may wait a day for a reviewer.
      timeout: Duration.minutes(1),
      durableConfig: {
        executionTimeout: reviewTimeout.plus(Duration.hours(1)),
        retentionPeriod: Duration.days(14),
      },
      environment: {
        TABLE_NAME: props.table.tableName,
        THUMBNAILER_FUNCTION: thumbnailer.currentVersion.functionArn,
        REVIEW_TIMEOUT: `${reviewTimeout.toSeconds()}s`,
      },
    });

    props.table.grantReadWriteData(workflow);
    props.uploads.grantRead(workflow); // Rekognition reads the image with the caller's permissions
    props.uploads.grantDelete(workflow); // rejected uploads are removed
    props.thumbnails.grantDelete(workflow);
    thumbnailer.grantInvoke(workflow);
    workflow.addToRolePolicy(
      new PolicyStatement({
        actions: ['rekognition:DetectLabels', 'rekognition:DetectModerationLabels'],
        resources: ['*'],
      }),
    );

    // A durable function is always invoked through a version or alias: an
    // execution that started on one version keeps replaying on that version,
    // even when new code is deployed while it waits for a reviewer.
    this.workflow = new Alias(this, 'Live', {
      aliasName: 'live',
      version: workflow.currentVersion,
    });
    this.thumbnailer = thumbnailer;

    // S3 starts an execution for every upload. No trigger function in between.
    props.uploads.addEventNotification(EventType.OBJECT_CREATED, new LambdaDestination(this.workflow), {
      prefix: USER_PREFIX,
    });
  }
}
