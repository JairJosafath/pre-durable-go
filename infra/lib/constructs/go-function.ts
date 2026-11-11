import path from 'path';
import { GoFunction, GoFunctionProps } from '@aws-cdk/aws-lambda-go-alpha';
import { Architecture, Runtime } from 'aws-cdk-lib/aws-lambda';
import { Construct } from 'constructs';

/** Root of the Go module: the repository root, one level above infra/. */
const moduleDir = path.join(__dirname, '..', '..', '..');

export type GoLambdaProps = Omit<GoFunctionProps, 'entry' | 'moduleDir' | 'runtime' | 'architecture'>;

/**
 * Builds the Go program in cmd/<command> into a Lambda function.
 *
 * Every function in this project is built the same way: a static arm64
 * binary named `bootstrap` on the OS-only runtime, as the durable execution
 * SDK for Go documents. `cdk synth` compiles with the local Go toolchain and
 * falls back to Docker when Go is not installed.
 */
export function goLambda(scope: Construct, id: string, command: string, props: GoLambdaProps = {}): GoFunction {
  return new GoFunction(scope, id, {
    entry: path.join(moduleDir, 'cmd', command),
    moduleDir,
    runtime: Runtime.PROVIDED_AL2023,
    architecture: Architecture.ARM_64,
    ...props,
    bundling: {
      cgoEnabled: false,
      goBuildFlags: ['-tags lambda.norpc', '-trimpath'],
      ...props.bundling,
    },
  });
}
