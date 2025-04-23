# Sync users from Okta

## Setup Okta SCIM application

1. Login to your okta admin console, go to Applications => Browse App Catalog, search for `SCIM 2.0 Test App (Basic Auth)` => Add Integration
2. On the General Settings tab, set application label and click Next
3. On the Sign-On Options you can leave all default values, scroll down and click Done
4. On the application page, go to the Provisioning tab => Configure API Integration => Enable API Integration
    - Set the SCIM 2.0 Base Url to https://{scim-endpoint}/
    - Set Username to your configured username
    - Set Password to your configured password
    - Test API Credentials
    - Save
5. Back on the Provisioning tab, on the To App Settings, click Edit, enable Create Users, Update User Attributes and Deactivate Users and Save

## Provision users

For provisoning users, a user needs to be assigned to the SCIM application.
1. Go to the Assignments tab
2. Click on Assign => Assign to People => Assign wanted users and click Done.
Your user should show up in the Directory
Any updates to a property that is mapped to a SCIM attribute, should trigger a user update in Aserto.

## Provision groups

For provisoning groups, a group needs to be assigned to the SCIM application.
1. Go to the Assignments tab
2. Click on Assign => Assign to People => Assign your group and click Done.
3. Go to Push Groups tab => Push Groups => Find groups by name => search for your group and click Save

Groups and group membership should be provisioned now.

## Troubleshooting
Please note that any errors on provisioning groups will pause the group provisioning. If a group was provisioned, Okta does keep a state for that provisioned group, so removing it from Aserto before attempting to unlink it from the Okta app can cause issues. If this happens, the group needs to be unlinked and reassigned to the app.
