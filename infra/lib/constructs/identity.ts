import { RemovalPolicy } from 'aws-cdk-lib/core';
import { DateTimeAttribute, StringAttribute, UserPool, UserPoolClient, VerificationEmailStyle } from 'aws-cdk-lib/aws-cognito';
import { Construct } from 'constructs';

/** Cognito user pool and the app client the web frontend signs in with. */
export class Identity extends Construct {
  readonly userPool: UserPool;
  readonly client: UserPoolClient;

  constructor(scope: Construct, id: string) {
    super(scope, id);

    this.userPool = new UserPool(this, 'UserPool', {
      selfSignUpEnabled: true,
      signInAliases: { username: true, email: true },
      signInCaseSensitive: false,
      autoVerify: { email: true },
      userVerification: {
        emailSubject: 'Welcome to PrimeDrive! Verify your email',
        emailBody: 'Hey {username}! Your PrimeDrive verification code is {####}',
        emailStyle: VerificationEmailStyle.CODE,
      },
      customAttributes: {
        subscription: new StringAttribute({ mutable: true }),
        joinedOn: new DateTimeAttribute({ mutable: true }),
      },
      removalPolicy: RemovalPolicy.DESTROY,
    });

    this.client = this.userPool.addClient('WebClient', {
      authFlows: { userPassword: true, userSrp: true },
    });
  }
}
