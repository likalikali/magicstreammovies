import { useState, useEffect } from 'react'
import './App.css'
import Home from './components/home/Home.jsx'
import Header from './components/header/Header.jsx'
import Register from './components/register/Register.jsx'
import Login from './components/login/Login.jsx'
import Review from './components/review/Review.jsx'
import {Route, Routes, useNavigate} from 'react-router-dom';
import Layout from './components/Layout.jsx'
import RequiredAuth from './components/RequiredAuth.jsx'
import Recommended from './components/recommend/Recommend.jsx'
import useAuth from './hook/useAuth';
import axiosClient from './api/axiosConfig';
import StreamMovie from './components/stream/StreamMovie.jsx';

function App() {
  const navigate = useNavigate();
  const { auth, setAuth } = useAuth();
  const updateMovieReview = (imdb_id) => {
    navigate(`/review/${imdb_id}`);
};
const handleLogout = async () => {

  try {
      const response = await axiosClient.post("/logout",{user_id: auth.user_id});
      console.log(response.data);
      setAuth(null);
     // localStorage.removeItem('user');
      console.log('User logged out');

  } catch (error) {
      console.error('Error logging out:', error);
  } 

};

  return (
    <>
      <Header handleLogout = {handleLogout}/>
      <Routes path="/" element={<Layout/>}>
        <Route path='/' element={<Home  updateMovieReview={updateMovieReview}/>}></Route>
        <Route path='/register' element={<Register/>}></Route>
        <Route path='/login' element={<Login/>}></Route>
        <Route element={<RequiredAuth/>}>
          <Route path='/recommended' element={<Recommended/>}></Route>
          <Route path='/review/:imdb_id' element={<Review/>}></Route>
          <Route path='/stream/:yt_id' element={<StreamMovie/>}></Route>
        </Route>
      </Routes>

    </>
  )
}

export default App
