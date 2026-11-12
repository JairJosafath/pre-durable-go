import { RemovalPolicy } from 'aws-cdk-lib/core';
import { BlockPublicAccess, Bucket, HttpMethods } from 'aws-cdk-lib/aws-s3';
import { Construct } from 'constructs';

/** Key prefix every user upload lives under: users/<cognito sub>/<file name>. */
export const USER_PREFIX = 'users/';

/** Buckets for the original uploads and for the thumbnails derived from them. */
export class Storage extends Construct {
  readonly uploads: Bucket;
  readonly thumbnails: Bucket;

  constructor(scope: Construct, id: string) {
    super(scope, id);

    this.uploads = new Bucket(this, 'Uploads', {
      blockPublicAccess: BlockPublicAccess.BLOCK_ALL,
      enforceSSL: true,
      removalPolicy: RemovalPolicy.DESTROY,
      autoDeleteObjects: true,
      // The browser uploads and downloads originals with presigned URLs.
      cors: [
        {
          allowedOrigins: ['*'],
          allowedMethods: [HttpMethods.GET, HttpMethods.PUT, HttpMethods.HEAD],
          allowedHeaders: ['*'],
          exposedHeaders: ['ETag'],
          maxAge: 3000,
        },
      ],
    });

    this.thumbnails = new Bucket(this, 'Thumbnails', {
      blockPublicAccess: BlockPublicAccess.BLOCK_ALL,
      enforceSSL: true,
      removalPolicy: RemovalPolicy.DESTROY,
      autoDeleteObjects: true,
    });
  }
}
