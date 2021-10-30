### How to use
- Clone the repo
- Make a copy of example-config.yml and rename it to config.yml
- Make sure your cloudflare token gives has authorization to edit dns zones
- Set a custom timeout (this aspect is about to be reworked, so watch out if you upgrade)
- Keep in mind this service only patches the content of records that already exist and that match the zones and names you provided (any provided name or zone that you don't have access to will not be touched, nor will this generate an error or unnecessary api calls)
- ???????
- profit.
