// index.ts Used in development for hot reloading/nodemon
import fs from 'fs'

fs.readdirSync(`${__dirname}/functions`)
  .filter((file) => file.slice(-3) === '.ts')
  .forEach((file) => {
    import(`./functions/${file}`)
  })
