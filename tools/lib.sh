# Shared by the scripts in tools/. Source it; do not run it.
#
# The scripts talk to the stack that `make deploy` created. They find it by
# name, in $AWS_REGION, and read everything else from the stack itself.

STACK=${PRIMEDRIVE_STACK:-PrimeDriveStack}
export AWS_REGION=${AWS_REGION:-eu-west-1}
export AWS_DEFAULT_REGION=$AWS_REGION

die() {
  echo "error: $*" >&2
  exit 1
}

# need <program>... stops when a program the script uses is not installed.
need() {
  local program
  for program in "$@"; do
    command -v "$program" >/dev/null || die "$program is not installed"
  done
}

# found <value> reports whether the AWS CLI printed a value rather than nothing.
found() {
  [ -n "$1" ] && [ "$1" != "None" ]
}

not_deployed() {
  die "no $STACK in $AWS_REGION that your AWS credentials can read. Deploy it with: make deploy"
}

# stack_output <key> prints an output of the deployed stack, such as ApiUrl.
stack_output() {
  local value
  value=$(aws cloudformation describe-stacks --stack-name "$STACK" \
    --query "Stacks[0].Outputs[?OutputKey=='$1'].OutputValue | [0]" --output text) || not_deployed
  found "$value" || die "$STACK has no output $1. Deploy the current code with: make deploy"
  printf '%s\n' "$value"
}

# stack_resource <type> prints the ID of the stack's resource of that type,
# such as AWS::Cognito::UserPool.
stack_resource() {
  local value
  value=$(aws cloudformation describe-stack-resources --stack-name "$STACK" \
    --query "StackResources[?ResourceType=='$1'].PhysicalResourceId | [0]" --output text) || not_deployed
  found "$value" || die "$STACK has no $1"
  printf '%s\n' "$value"
}

# stack_resource_named <prefix> prints the ID of the stack's resource whose
# logical name starts with the prefix, such as StorageThumbnails. For when
# the stack has several resources of one type.
stack_resource_named() {
  local value
  value=$(aws cloudformation describe-stack-resources --stack-name "$STACK" \
    --query "StackResources[?starts_with(LogicalResourceId, '$1')].PhysicalResourceId | [0]" --output text) || not_deployed
  found "$value" || die "$STACK has no resource named $1..."
  printf '%s\n' "$value"
}
