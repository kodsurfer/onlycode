import express from 'express';
//import authRoutes from './auth.js';
//import reposRoutes from './repositories.js';
//import reviewRoutes from './reviews.js';
//import userRoutes from './users.js';

export const setupRoutes = (app) => {
  app.use('/api/auth', authRoutes);
  app.use('/api/repos', reposRoutes);
  app.use('/api/reviews', reviewRoutes);
  app.use('/api/users', userRoutes);
};
