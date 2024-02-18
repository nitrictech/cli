# Nitric Local Dashboard

<p align="center">
  <a href="https://nitric.io">
    <img src="../docs/assets/nitric-logo.svg" width="120" alt="Nitric Logo"/>
  </a>
</p>

<p align="center">
  CLI for building and deploying <a href="https://nitric.io">nitric</a> apps
</p>

## 🚀 Project Structure

Inside the dashboard project, you'll see the following folders and files:

```
/
├── public/
│   └── favicon.ico
├── src/
│   ├── components/
│   ├── layouts/
|   ├── lib/
│   └── pages/
│       └── index.astro
└── package.json
```

Astro looks for `.astro` or `.md` files in the `src/pages/` directory. Each page is exposed as a route based on its file name.

There's nothing special about `src/components/`, but that's where we like to put any Astro or React components.

Any static assets, like images, can be placed in the `public/` directory.

## 🧞 Commands

All commands are run from the root of the project, from a terminal:

| Command             | Action                                                    |
| :------------------ | :-------------------------------------------------------- |
| `yarn install`      | Installs dependencies                                     |
| `yarn dev`          | Starts local dev server at `localhost:3000`               |
| `yarn build`        | Build the production dashboard to `../pkg/dashboard/dist` |
| `yarn preview`      | Preview your build locally, before deploying              |
| `yarn cypress:open` | Open Cypress for e2e testing                              |
| `yarn astro ...`    | Run CLI commands like `astro add`, `astro check`          |
| `yarn astro --help` | Get help using the Astro CLI                              |

## Need help with Nitric?

Feel free to check out the [Nitric documentation](https://nitric.io/docs) or jump on our [Discord Server](https://nitric.io/chat).
